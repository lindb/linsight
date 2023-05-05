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
	"database/sql"
	"errors"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"github.com/lindb/linsight/config"
)

type User struct {
	ID        int64     `gorm:"column:id"`
	Name      string    `gorm:"column:name"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`

	CreatedBy int64 `gorm:"column:created_by"`
	UpdatedBy int64 `gorm:"column:updated_by"`
}

func TestDB_initialize(t *testing.T) {
	cases := []struct {
		name    string
		prepare func() *db
		wantErr bool
	}{
		{
			name: "not support db",
			prepare: func() *db {
				return &db{
					cfg: &config.Database{
						Type: "oracle",
					},
				}
			},
			wantErr: true,
		},
		{
			name: "sqlite",
			prepare: func() *db {
				return &db{
					cfg: &config.Database{
						DSN:  filepath.Join(t.TempDir(), "db.db"),
						Type: "sqlite",
					},
				}
			},
			wantErr: false,
		},
		{
			name: "init mysql failure",
			prepare: func() *db {
				conn, _, err := sqlmock.New()
				assert.NoError(t, err)
				return &db{
					cfg: &config.Database{
						DSN:   "lin:admin@tcp(localhost:3306)/linsight",
						Type:  "mysql",
						Debug: true,
					},
					conn: conn,
				}
			},
			wantErr: true,
		},
		{
			name: "init mysql successfully",
			prepare: func() *db {
				conn, _, err := sqlmock.New()
				assert.NoError(t, err)
				return &db{
					cfg: &config.Database{
						DSN:   "lin:admin@tcp(localhost:3306)/linsight",
						Type:  "mysql",
						Debug: true,
					},
					conn:                      conn,
					skipInitializeWithVersion: true,
				}
			},
			wantErr: false,
		},
		{
			name: "init postgres successfully",
			prepare: func() *db {
				conn, _, err := sqlmock.New()
				assert.NoError(t, err)
				return &db{
					cfg: &config.Database{
						DSN:   "lin:admin@tcp(localhost:3306)/linsight",
						Type:  "postgres",
						Debug: true,
					},
					conn: conn,
				}
			},
			wantErr: false,
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			db := tt.prepare()
			err := db.initialize()
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestDB_NewDB(t *testing.T) {
	db, err := NewDB(&config.Database{
		DSN:  filepath.Join(t.TempDir(), "db.db"),
		Type: "sqlite",
	})
	assert.NoError(t, err)
	assert.NotNil(t, db)
	assert.NotNil(t, db.RawDB())
	err = db.Close()
	assert.NoError(t, err)
	db, err = NewDB(&config.Database{
		DSN:  filepath.Join(t.TempDir(), "db.db"),
		Type: "oracle",
	})
	assert.Error(t, err)
	assert.Nil(t, db)
}

func TestDB_Create(t *testing.T) {
	database, mock := GetDBMock(t)
	cases := []struct {
		name    string
		prepare func(mock sqlmock.Sqlmock)
		assert  func(user *User, err error)
	}{
		{
			name: "create successfully",
			prepare: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("INSERT INTO `users`").
					WithArgs("test", sqlmock.AnyArg(), sqlmock.AnyArg(), int64(1), int64(2)).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			assert: func(user *User, err error) {
				assert.NoError(t, err)
				assert.Equal(t, int64(1), user.ID)
			},
		},
		{
			name: "create fauilre",
			prepare: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("INSERT INTO `users`").
					WithArgs("test", sqlmock.AnyArg(), sqlmock.AnyArg(), int64(1), int64(2)).
					WillReturnError(errors.New("err"))
				mock.ExpectRollback()
			},
			assert: func(_ *User, err error) {
				assert.Equal(t, errors.New("err").Error(), err.Error())
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			expectValue := &User{
				Name:      "test",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				CreatedBy: 1,
				UpdatedBy: 2,
			}
			tt.prepare(mock)
			err := database.Create(expectValue)
			tt.assert(expectValue, err)
		})
	}
}

func TestDB_Update(t *testing.T) {
	database, mock := GetDBMock(t)
	cases := []struct {
		name    string
		prepare func(mock sqlmock.Sqlmock)
		assert  func(err error)
	}{
		{
			name: "update successfully",
			prepare: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE `users`").
					WithArgs("test", sqlmock.AnyArg(), "test").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			assert: func(err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "update fauilre",
			prepare: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE `users`").
					WithArgs("test", sqlmock.AnyArg(), "test").
					WillReturnError(errors.New("err"))
				mock.ExpectRollback()
			},
			assert: func(err error) {
				assert.Equal(t, errors.New("err").Error(), err.Error())
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			expectValue := &User{
				Name: "test",
			}
			tt.prepare(mock)
			err := database.Update(expectValue, "name=?", "test")
			tt.assert(err)
		})
	}
}

func TestDB_Find(t *testing.T) {
	database, mock := GetDBMock(t)
	var users []User
	rows := sqlmock.NewRows([]string{"name"})
	rows.AddRow("test")
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `users` WHERE name = ?")).
		WithArgs("test").
		WillReturnRows(rows)
	err := database.Find(&users, "name = ?", "test")
	assert.NoError(t, err)
	assert.Len(t, users, 1)
}

func TestDB_FindWithPaging(t *testing.T) {
	database, mock := GetDBMock(t)
	var users []User
	rows := sqlmock.NewRows([]string{"name"})
	rows.AddRow("test")
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `users` WHERE name = ?")).
		WithArgs("test").
		WillReturnRows(rows)
	err := database.FindForPaging(&users, 0, 10, "id desc", "name = ?", "test")
	assert.NoError(t, err)
	assert.Len(t, users, 1)
}

func TestDB_Count(t *testing.T) {
	database, mock := GetDBMock(t)
	expectSQL := "SELECT count(*) FROM `users` WHERE name = ?"
	condition := &User{
		Name: "xx",
	}
	t.Run("count ok", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(expectSQL)).WithArgs(condition.Name).
			WillReturnRows(sqlmock.NewRows([]string{"count(1)"}).AddRow(10))
		count, err := database.Count(&User{}, "name = ?",
			condition.Name)
		assert.NoError(t, err)
		assert.Equal(t, int64(10), count)
	})
	t.Run("return err", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(expectSQL)).WithArgs(condition.Name).
			WillReturnError(errors.New("xx"))
		count, err := database.Count(&User{}, "name = ?",
			condition.Name)
		assert.Error(t, err)
		assert.Equal(t, int64(0), count)
	})
}

func TestDB_Get(t *testing.T) {
	database, mock := GetDBMock(t)
	var user User
	rows := sqlmock.NewRows([]string{"name"})
	rows.AddRow("test")
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `users` WHERE name = ?")).
		WithArgs("test").
		WillReturnRows(rows)
	err := database.Get(&user, "name = ?", "test")
	assert.NoError(t, err)
	assert.Equal(t, "test", user.Name)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `users` WHERE name = ?")).
		WithArgs("test").
		WillReturnError(errors.New("err"))
	err = database.Get(&user, "name = ?", "test")
	assert.Error(t, err)
}

func TestDB_Exist(t *testing.T) {
	database, mock := GetDBMock(t)
	cases := []struct {
		name    string
		prepare func(mock sqlmock.Sqlmock)
		assert  func(exist bool, err error)
	}{
		{
			name: "not found",
			prepare: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT").
					WithArgs("test").
					WillReturnError(gorm.ErrRecordNotFound)
			},
			assert: func(exist bool, err error) {
				assert.False(t, exist)
				assert.NoError(t, err)
			},
		},
		{
			name: "check failure",
			prepare: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT").
					WithArgs("test").
					WillReturnError(errors.New("err"))
			},
			assert: func(exist bool, err error) {
				assert.False(t, exist)
				assert.Error(t, err)
			},
		},
		{
			name: "resource found",
			prepare: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"name"})
				rows.AddRow("test")
				mock.ExpectQuery("SELECT").
					WithArgs("test").
					WillReturnRows(rows)
			},
			assert: func(exist bool, err error) {
				assert.True(t, exist)
				assert.NoError(t, err)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			tt.prepare(mock)
			exist, err := database.Exist(&User{}, "name=?", "test")
			tt.assert(exist, err)
		})
	}
}

func TestDB_Delete(t *testing.T) {
	database, mock := GetDBMock(t)
	var user User
	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM `users` WHERE name = ?").
		WithArgs("test").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	err := database.Delete(&user, "name = ?", "test")
	assert.NoError(t, err)
}

func GetDBMock(t *testing.T) (DB, sqlmock.Sqlmock) {
	var err error
	var conn *sql.DB
	conn, mock, err := sqlmock.New()
	assert.NoError(t, err)
	database := &db{
		cfg: &config.Database{
			DSN:   "lin:admin@tcp(localhost:3306)/linsight",
			Type:  "mysql",
			Debug: true,
		},
		conn:                      conn,
		skipInitializeWithVersion: true,
	}
	err = database.initialize()
	if err != nil {
		t.Fatal("cannot initialize database")
	}
	return database, mock
}
