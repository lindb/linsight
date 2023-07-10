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
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/lindb/linsight/pkg/db/gorm"
)

func TestMigration(t *testing.T) {
	m := NewMigration(&User{})
	m.AddInitRecord(&User{}, "id=?", 123)
	assert.Len(t, m.records, 1)
	assert.Equal(t, &User{}, m.table)
}

func TestMigrator(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	migration := NewMigration(&User{})
	migration.AddInitRecord(&User{}, "id=?", 123)

	mockM := gorm.NewMockMigrator(ctrl)
	mockDB := NewMockDB(ctrl)
	mockDB.EXPECT().Migrator().Return(mockM).AnyTimes()

	migrator := NewMigrator(mockDB)
	migrator.AddMigration(migration)

	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "auto migrate failure",
			prepare: func() {
				mockM.EXPECT().HasTable(gomock.Any()).Return(true)
				mockM.EXPECT().AutoMigrate(gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "create table failure",
			prepare: func() {
				mockM.EXPECT().HasTable(gomock.Any()).Return(false)
				mockM.EXPECT().CreateTable(gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "check init record failure",
			prepare: func() {
				mockM.EXPECT().HasTable(gomock.Any()).Return(false)
				mockM.EXPECT().CreateTable(gomock.Any()).Return(nil)
				mockDB.EXPECT().Exist(gomock.Any(), "id=?", 123).Return(false, fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "create init record failure",
			prepare: func() {
				mockM.EXPECT().HasTable(gomock.Any()).Return(false)
				mockM.EXPECT().CreateTable(gomock.Any()).Return(nil)
				mockDB.EXPECT().Exist(gomock.Any(), "id=?", 123).Return(false, nil)
				mockDB.EXPECT().Create(gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "update init record failure",
			prepare: func() {
				mockM.EXPECT().HasTable(gomock.Any()).Return(false)
				mockM.EXPECT().CreateTable(gomock.Any()).Return(nil)
				mockDB.EXPECT().Exist(gomock.Any(), "id=?", 123).Return(true, nil)
				mockDB.EXPECT().Updates(gomock.Any(), gomock.Any(), "id=?", 123).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "run successfully",
			prepare: func() {
				mockM.EXPECT().HasTable(gomock.Any()).Return(false)
				mockM.EXPECT().CreateTable(gomock.Any()).Return(nil)
				mockDB.EXPECT().Exist(gomock.Any(), "id=?", 123).Return(true, nil)
				mockDB.EXPECT().Updates(gomock.Any(), gomock.Any(), "id=?", 123).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			err := migrator.Run()
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}
