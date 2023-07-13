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

	"github.com/lindb/linsight"
	"github.com/lindb/linsight/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
)

//go:generate mockgen -source=./integration.go -destination=./integration_mock.go -package=service

// IntegrationService represents platform integration manager interface.
type IntegrationService interface {
	// Initialize initializes supported platform integrations.
	Initialize() error
	// GetIntegrations returns all supported integrations.
	GetIntegrations(ctx context.Context) ([]model.Integration, error)
	// ConnectSource connects integration and source.
	ConnectSource(ctx context.Context, integrationUID, sourceUID string, connectionType model.ConnectionType) error
	// DisconnectSource disconnects integration and source.
	DisconnectSource(ctx context.Context, sourceUID string, connectionType model.ConnectionType) error
}

type integrationService struct {
	db dbpkg.DB
}

// NewIntegrationService creates an IntegrationService instance.
func NewIntegrationService(db dbpkg.DB) IntegrationService {
	return &integrationService{
		db: db,
	}
}

// Initialize initializes supported platform integrations.
func (srv *integrationService) Initialize() error {
	var integrations []model.Integration
	if err := jsonUnmarshalFn([]byte(linsight.Integrationss), &integrations); err != nil {
		return err
	}
	for idx := range integrations {
		integration := integrations[idx]
		exist, err := srv.db.Exist(&model.Integration{}, "uid=?", integration.UID)
		if err != nil {
			return err
		}
		if exist {
			// update
			if err0 := srv.db.Update(&integration, "uid=?", integration.UID); err0 != nil {
				return err0
			}
		} else {
			// create
			if err0 := srv.db.Create(&integration); err0 != nil {
				return err0
			}
		}
	}
	return nil
}

// GetIntegrations returns all supported integrations.
func (srv *integrationService) GetIntegrations(ctx context.Context) ([]model.Integration, error) {
	var integrations []model.Integration
	if err := srv.db.Find(&integrations); err != nil {
		return nil, err
	}
	return integrations, nil
}

// ConnectSource connects integration and source.
func (srv *integrationService) ConnectSource(ctx context.Context,
	integrationUID, sourceUID string,
	connectionType model.ConnectionType) error {
	signedUser := util.GetUser(ctx)
	return srv.db.Transaction(func(tx dbpkg.DB) error {
		if err := tx.Delete(&model.IntegrationConnection{},
			"org_id=? and source_uid=? and type=?",
			signedUser.Org.ID, sourceUID, connectionType); err != nil {
			return err
		}
		return tx.Create(&model.IntegrationConnection{
			BaseModel: model.BaseModel{
				CreatedBy: signedUser.User.ID,
				UpdatedBy: signedUser.User.ID,
			},
			OrgID:          signedUser.Org.ID,
			IntegrationUID: integrationUID,
			SourceUID:      sourceUID,
			Type:           connectionType,
		})
	})
}

// DisconnectSource disconnects integration and source.
func (srv *integrationService) DisconnectSource(ctx context.Context,
	sourceUID string, connectionType model.ConnectionType) error {
	signedUser := util.GetUser(ctx)
	return srv.db.Delete(&model.IntegrationConnection{},
		"org_id=? and source_uid=? and type=?",
		signedUser.Org.ID, sourceUID, connectionType)
}
