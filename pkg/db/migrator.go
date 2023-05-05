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

type record struct {
	value any
	where []any
}

type Migration struct {
	table   any
	records []*record
}

func NewMigration(table any) *Migration {
	return &Migration{
		table: table,
	}
}

func (m *Migration) AddInitRecord(value any, where ...any) {
	m.records = append(m.records, &record{
		value: value,
		where: where,
	})
}

type Migrator interface {
	AddMigration(migration *Migration)
	Run() error
}

type migrator struct {
	db         DB
	migrations []*Migration
}

func NewMigrator(db DB) Migrator {
	return &migrator{
		db: db,
	}
}

func (m *migrator) AddMigration(migration *Migration) {
	m.migrations = append(m.migrations, migration)
}

func (m *migrator) Run() error {
	var err error
	migrator := m.db.RawDB().Migrator()
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

		for _, record := range migration.records {
			ok, err := m.db.Exist(record.value, record.where...)
			if err != nil {
				return err
			}
			if ok {
				err = m.db.RawDB().Updates(record.value).Error
				if err != nil {
					return err
				}
			} else {
				err = m.db.Create(record.value)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
