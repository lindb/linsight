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

package http

import (
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin/binding"
	"github.com/lindb/common/pkg/encoding"
)

// QueryJSONBind represents url query params json bind.
var QueryJSONBind binding.Binding

func init() {
	QueryJSONBind = QueryJSONBinder{}
}

// QueryJSONBinder represents url query params json binder.
// implements binding.Binding interface.
type QueryJSONBinder struct{}

// Name returns the binder's name.
func (QueryJSONBinder) Name() string {
	return "queryJSONBinder"
}

// Bind binds request data to obj.
func (QueryJSONBinder) Bind(req *http.Request, obj any) error {
	values := req.URL.Query()
	if len(values) == 0 || obj == nil {
		return nil
	}
	jsonData := make(map[string]any)

	val := reflect.ValueOf(obj).Elem()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		fieldType := typ.Field(i)
		strValue := fieldType.Tag.Get("form")
		if fieldType.Type.Kind() == reflect.Slice || fieldType.Type.Kind() == reflect.Array {
			jsonData[strValue] = values[strValue]
		} else {
			jsonData[strValue] = values.Get(strValue)
		}
	}

	jsonBytes := encoding.JSONMarshal(jsonData)
	return encoding.JSONUnmarshal(jsonBytes, obj)
}
