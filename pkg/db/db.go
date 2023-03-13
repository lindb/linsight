// Licensed to LinDB under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. LinDB licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package db

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/lindb/linsight/config"
)

//go:generate mockgen -source=./db.go -destination=./db_mock.go -package=db

// DB represents a db wrapper based on gorm.
// Support: sqlit/mysql/postgres.
type DB interface {
	io.Closer

	// Create creates record.
	Create(obj any) error
	// Update updates record by given conditions.
	Update(obj any, where ...any) error
	// Delete deletes record by given conditions.
	Delete(obj any, where ...any) error
	// Get gets record that match given conditions.
	Get(out any, where ...any) error
	// Find finds records that match given conditions.
	Find(out any, where ...any) error
	// FindForPaging finds records that match given conditions with offset and limit.
	FindForPaging(out any, offset, limit int, order string, query string, where ...any) error
	// Count returns how many records for a model based on given query conditions.
	Count(model any, query string, where ...any) (count int64, err error)
	// Exist checks record if exist by given conditions.
	Exist(out any, where ...any) (bool, error)
	// Transaction does some db operators in transaction.
	Transaction(fc func(tx DB) error) error
	// RawDB returns raw gorm db.
	RawDB() *gorm.DB
}

// db implements DB interface.
type db struct {
	gdb *gorm.DB
	cfg *config.Database

	// just for test
	conn                      gorm.ConnPool
	skipInitializeWithVersion bool
}

// NewDB creates a DB instance.
func NewDB(cfg *config.Database) (DB, error) {
	db := &db{
		cfg: cfg,
	}
	err := db.initialize()
	if err != nil {
		return nil, err
	}
	return db, nil
}

func newWrapper(gdb *gorm.DB) DB {
	return &db{gdb: gdb}
}

// initialize initialize(s database based on type/configuration.)
func (db *db) initialize() error {
	dbType := db.cfg.Type
	// FIXME: add more config options
	var dialector gorm.Dialector
	switch dbType {
	case "sqlite":
		dialector = sqlite.Open(db.cfg.DSN)
	case "mysql":
		dialector = mysql.New(mysql.Config{
			DSN:                       db.cfg.DSN,
			DriverName:                "mysql",
			Conn:                      db.conn,
			SkipInitializeWithVersion: db.skipInitializeWithVersion,
		})
	case "postgres":
		dialector = postgres.New(postgres.Config{
			DSN:        db.cfg.DSN,
			DriverName: "postgres",
			Conn:       db.conn,
		})
	default:
		return fmt.Errorf("unknown database type, type:%s", dbType)
	}
	gdb, err := gorm.Open(dialector, &gorm.Config{
		IgnoreRelationshipsWhenMigrating: true,
	})
	if err != nil {
		return err
	}
	if db.cfg.Debug {
		gdb = gdb.Debug()
	}
	db.gdb = gdb
	return nil
}

// Create creates record.
func (db *db) Create(obj any) error {
	err := db.gdb.Create(obj).Error
	return errors.WithStack(err)
}

// Update updates record by given conditions.
func (db *db) Update(obj any, where ...any) error {
	_, err := db.UpdateWithResult(obj, where...)
	return err
}

// UpdateWithResult updates record by given conditions, then returns the rows affected.
func (db *db) UpdateWithResult(obj any, where ...any) (int64, error) {
	m := db.gdb.Model(obj)
	if len(where) > 0 {
		m = m.Where(where[0], where[1:]...)
	}
	m = m.Updates(obj)
	return m.RowsAffected, errors.WithStack(m.Error)
}

// Find finds records that match given conditions.
func (db *db) Find(out any, where ...any) error {
	err := db.gdb.Find(out, where...).Error
	return errors.WithStack(err)
}

// FindForPaging finds records that match given conditions with offset and limit.
func (db *db) FindForPaging(out any,
	offset, limit int, order string,
	query string, where ...any,
) error {
	err := db.gdb.Order(order).Offset(offset).Limit(limit).Where(query, where...).Find(out).Error
	return errors.WithStack(err)
}

// Count returns how many records for a model based on given query conditions.
func (db *db) Count(model any, query string, where ...any) (count int64, err error) {
	err = db.gdb.Model(model).Where(query, where...).Count(&count).Error
	if err != nil {
		err = errors.WithStack(err)
	}
	return
}

// Exist checks record if exist by given conditions.
func (db *db) Exist(out any, where ...any) (bool, error) {
	res := db.gdb.First(out, where...)
	// check err is record not found
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if res.Error != nil {
		return false, errors.WithStack(res.Error)
	}
	return true, nil
}

// Get gets record that match given conditions.
func (db *db) Get(out any, where ...any) error {
	err := db.gdb.First(out, where...).Error
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// Delete deletes record by given conditions.
func (db *db) Delete(obj any, where ...any) error {
	return db.gdb.Delete(obj, where...).Error
}

// Transaction does some db operators in transaction.
func (db *db) Transaction(fc func(tx DB) error) error {
	var err error
	panicked := true
	trans := db.gdb.Begin()
	defer func() {
		if panicked || err != nil {
			trans.Rollback()
		}
	}()
	err = fc(newWrapper(trans))
	if err != nil {
		return errors.WithStack(err)
	}
	err = trans.Commit().Error
	panicked = false
	return err
}

// RawDB returns raw gorm db.
func (db *db) RawDB() *gorm.DB {
	return db.gdb
}

// Close closes db, then waits for all queries that have started processing on the server.
func (db *db) Close() error {
	d, err := db.gdb.DB()
	if err != nil {
		return err
	}
	return d.Close()
}
