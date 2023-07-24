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

package model

import (
	"testing"

	"github.com/lindb/common/pkg/encoding"
	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"
)

func TestDashboard_ReadMeta(t *testing.T) {
	dashboard := &Dashboard{
		Config: datatypes.JSON(`{"title": "John", "description": "desc", "uid":"UID"}`),
	}
	dashboard.ReadMeta()
	assert.Equal(t, "John", dashboard.Title)
	assert.Equal(t, "desc", dashboard.Desc)
	assert.Equal(t, "UID", dashboard.UID)
}

func TestDashboard_PermissionType_String(t *testing.T) {
	assert.Equal(t, "Member", PermissionMember.String())
	assert.Equal(t, "Admin", PermissionAdmin.String())
	assert.Equal(t, "Unknown", PermissionUnknown.String())
	assert.Equal(t, "Unknown", PermissionType(9999).String())
}

func TestDashboard_Permission_JSOM(t *testing.T) {
	defer func() {
		jsonUnmarshalFn = encoding.JSONUnmarshal
	}()
	data, err := PermissionMember.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, []byte(`"Member"`), data)

	p := PermissionAdmin
	err = (&p).UnmarshalJSON(data)
	assert.NoError(t, err)
	assert.Equal(t, PermissionMember, p)

	err = (&p).UnmarshalJSON([]byte("abc"))
	assert.Error(t, err)

	err = (&p).UnmarshalJSON([]byte(`"abc"`))
	assert.NoError(t, err)
	assert.Equal(t, PermissionUnknown, p)
}

func TestDashboard_GetCharts(t *testing.T) {
	cases := []struct {
		name      string
		dashboard *Dashboard
		wantErr   bool
		chartIDs  []string
	}{
		{
			name: "panels not json array",
			dashboard: &Dashboard{
				Config: datatypes.JSON(`{"panels":"abc"}`),
			},
			wantErr: false,
		},
		{
			name: "panels empty",
			dashboard: &Dashboard{
				Config: datatypes.JSON(`{"panels":[]}`),
			},
			wantErr: false,
		},
		{
			name: "no chart link",
			dashboard: &Dashboard{
				Config: datatypes.JSON(`{"panels":[{}]}`),
			},
			wantErr: false,
		},
		{
			name: "not chart uid",
			dashboard: &Dashboard{
				Config: datatypes.JSON(`{"panels":[{"libraryPanel":{}}]}`),
			},
			wantErr: true,
		},
		{
			name: "get chart uid",
			dashboard: &Dashboard{
				Config: datatypes.JSON(`{"panels":[{"libraryPanel":{"uid":"abc"}}]}`),
			},
			wantErr:  false,
			chartIDs: []string{"abc"},
		},
		{
			name: "row panel no panels",
			dashboard: &Dashboard{
				Config: datatypes.JSON(`{"panels":[{"type":"row"}]}`),
			},
			wantErr: false,
		},
		{
			name: "row panel miss chart id",
			dashboard: &Dashboard{
				Config: datatypes.JSON(`{"panels":[{"type":"row","panels":[{"libraryPanel":{}}]}]}`),
			},
			wantErr: true,
		},
		{
			name: "row panel miss chart id",
			dashboard: &Dashboard{
				Config: datatypes.JSON(`{"panels":[{"type":"row","panels":[{"libraryPanel":{"uid":"abc"}}]}]}`),
			},
			wantErr:  false,
			chartIDs: []string{"abc"},
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			chartIDs, err := tt.dashboard.GetCharts()
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
			assert.Equal(t, tt.chartIDs, chartIDs)
		})
	}
}

func TestDashboard_NewDashboardMeta(t *testing.T) {
	meta := NewDashboardMeta()
	assert.True(t, meta.CanEdit)
	assert.False(t, meta.Provisioned)
}
