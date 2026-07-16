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

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lindb/linsight/accesscontrol"
	agentapi "github.com/lindb/linsight/ai/api"
	"github.com/lindb/linsight/constant"
	"github.com/lindb/linsight/model"
)

// specialistCtx builds a request context carrying a signed org-admin user.
func specialistCtx() context.Context {
	return context.WithValue(context.TODO(), constant.LinSightSignedKey, &model.SignedUser{
		Org:  &model.Org{BaseModel: model.BaseModel{ID: 1}},
		User: &model.User{BaseModel: model.BaseModel{ID: 1}},
		Role: accesscontrol.RoleAdmin,
	})
}

// These paths all return before any DB access, so a nil DB is safe and keeps the test
// focused on the validation and authorization gates unique to SpecialistService.
func TestCreateSpecialist_Validation(t *testing.T) {
	srv := NewSpecialistService(nil)

	t.Run("missing name", func(t *testing.T) {
		_, err := srv.CreateSpecialist(specialistCtx(), &agentapi.AgentRequest{
			Content: "you are an expert",
		})
		assert.Error(t, err)
	})

	t.Run("missing content", func(t *testing.T) {
		_, err := srv.CreateSpecialist(specialistCtx(), &agentapi.AgentRequest{
			Name: "metrics-analyst",
		})
		assert.Error(t, err)
	})

	t.Run("invalid name slug", func(t *testing.T) {
		_, err := srv.CreateSpecialist(specialistCtx(), &agentapi.AgentRequest{
			Name:    "bad name!",
			Content: "you are an expert",
		})
		assert.Error(t, err)
	})
}

func TestCreateSpecialist_GlobalGate(t *testing.T) {
	srv := NewSpecialistService(nil)

	// A non-super-admin may not create a global specialist; the gate fires before any DB call.
	_, err := srv.CreateSpecialist(specialistCtx(), &agentapi.AgentRequest{
		Name:     "global-analyst",
		Content:  "you are an expert",
		IsGlobal: true,
	})
	assert.ErrorIs(t, err, ErrGlobalRuleForbidden)
}

func TestUpdateSpecialist_Validation(t *testing.T) {
	srv := NewSpecialistService(nil)

	_, err := srv.CreateSpecialist(specialistCtx(), &agentapi.AgentRequest{})
	assert.Error(t, err)

	// Update rejects an invalid name before touching the DB.
	err = srv.UpdateSpecialist(specialistCtx(), "uid-1", &agentapi.AgentRequest{
		Name:    "has space",
		Content: "x",
	})
	assert.Error(t, err)
}
