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
