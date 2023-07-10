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

//go:generate mockgen -destination=./gorm/migrator_mock.go -package=gorm gorm.io/gorm Migrator

// record represents init record.
type record struct {
	value any
	where []any
}

// Migration represents migration information include table and init records.
type Migration struct {
	table   any
	records []*record
}

// NewMigration creates a data migration.
func NewMigration(table any) *Migration {
	return &Migration{
		table: table,
	}
}

// AddInitRecord adds init record.
func (m *Migration) AddInitRecord(value any, where ...any) {
	m.records = append(m.records, &record{
		value: value,
		where: where,
	})
}

// Migrator represents data migrator interface.
type Migrator interface {
	// AddMigration adds migration data.
	AddMigration(migration *Migration)
	// Run runs data migration include table and data.
	Run() error
}

// migrator implements Migrator interface.
type migrator struct {
	db         DB
	migrations []*Migration
}

// NewMigrator creates a Migrator instance.
func NewMigrator(db DB) Migrator {
	return &migrator{
		db: db,
	}
}

// AddMigration adds migration data.
func (m *migrator) AddMigration(migration *Migration) {
	m.migrations = append(m.migrations, migration)
}

// Run runs data migration include table and data.
func (m *migrator) Run() error {
	var err error
	migrator := m.db.Migrator()
	for _, migration := range m.migrations {
		table := migration.table
		if migrator.HasTable(table) {
			err = migrator.AutoMigrate(table)
		} else {
			err = migrator.CreateTable(table)
		}
		if err != nil {
			return err
		}

		// handle init records if exist
		for _, record := range migration.records {
			ok, err := m.db.Exist(record.value, record.where...)
			if err != nil {
				return err
			}
			if ok {
				// if init record exist, update it
				err = m.db.Updates(record.value, record.value, record.where...)
				if err != nil {
					return err
				}
			} else {
				// create init record
				err = m.db.Create(record.value)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
