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
	"fmt"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/pkg/db"
)

// ptr helpers used in test data construction.
func ptrInt64(v int64) *int64 { return &v }
func ptrStr(v string) *string { return &v }
func ptrInt(v int) *int       { return &v }

// makeGroupRow builds a groupRuleRow that represents a group with no matched rule (LEFT JOIN NULL side).
func makeGroupRow(gid int64, guid, gname string, interval time.Duration) groupRuleRow {
	return groupRuleRow{
		GID:       gid,
		GUID:      guid,
		GName:     gname,
		GInterval: interval,
	}
}

// makeGroupRuleRow builds a groupRuleRow with both group and rule fields populated.
func makeGroupRuleRow(gid int64, guid, gname string, gInterval time.Duration,
	rid int64, ruid, rGroupUID, rTitle, rDatasourceUID string) groupRuleRow {
	row := makeGroupRow(gid, guid, gname, gInterval)
	row.RID = ptrInt64(rid)
	row.RUID = ptrStr(ruid)
	row.RGroupUID = ptrStr(rGroupUID)
	row.RTitle = ptrStr(rTitle)
	row.RDatasourceUID = ptrStr(rDatasourceUID)
	row.ROrgID = ptrInt64(12)
	row.RVersion = ptrInt(1)
	row.RState = ptrInt(1)
	row.RInterval = ptrInt64(int64(gInterval))
	row.RFor = ptrInt64(0)
	row.RCreatedBy = ptrInt64(10)
	row.RUpdatedBy = ptrInt64(10)
	return row
}

func TestAlertRuleService_SearchAlertRules(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewAlertRuleService(mockDB, nil)

	interval := 30 * time.Second

	cases := []struct {
		name       string
		req        *model.SearchAlertRuleRequest
		prepare    func()
		wantTotal  int64
		wantGroups int
		wantRules  []int // expected rule count per group (in result order)
		wantErr    bool
	}{
		{
			name: "db error",
			req:  &model.SearchAlertRuleRequest{},
			prepare: func() {
				mockDB.EXPECT().ExecRaw(gomock.Any(), gomock.Any(), int64(12)).
					Return(fmt.Errorf("db error"))
			},
			wantErr: true,
		},
		{
			name: "no rows — empty result",
			req:  &model.SearchAlertRuleRequest{},
			prepare: func() {
				mockDB.EXPECT().ExecRaw(gomock.Any(), gomock.Any(), int64(12)).
					DoAndReturn(func(out any, _ string, _ ...any) error {
						// out is *[]groupRuleRow; leave it empty (zero value already is nil slice)
						return nil
					})
			},
			wantTotal:  0,
			wantGroups: 0,
		},
		{
			name: "all groups with rules returned when no filters",
			req:  &model.SearchAlertRuleRequest{},
			prepare: func() {
				mockDB.EXPECT().ExecRaw(gomock.Any(), gomock.Any(), int64(12)).
					DoAndReturn(func(out any, _ string, _ ...any) error {
						rows := []groupRuleRow{
							makeGroupRuleRow(2, "g2", "infra", interval, 20, "r2", "g2", "disk-alert", "ds-1"),
							makeGroupRuleRow(1, "g1", "app", interval, 10, "r1", "g1", "cpu-alert", "ds-1"),
							makeGroupRuleRow(1, "g1", "app", interval, 11, "r3", "g1", "mem-alert", "ds-1"),
						}
						*(out.(*[]groupRuleRow)) = rows
						return nil
					})
			},
			wantTotal:  2,
			wantGroups: 2,
			wantRules:  []int{1, 2}, // g2→1 rule, g1→2 rules
		},
		{
			name: "groupName filter — group with no rules still returned",
			req:  &model.SearchAlertRuleRequest{GroupName: "infra"},
			prepare: func() {
				mockDB.EXPECT().ExecRaw(gomock.Any(), gomock.Any(), int64(12), "infra%").
					DoAndReturn(func(out any, _ string, _ ...any) error {
						// LEFT JOIN: group matches, but no rules exist for it
						rows := []groupRuleRow{
							makeGroupRow(2, "g2", "infra", interval),
						}
						*(out.(*[]groupRuleRow)) = rows
						return nil
					})
			},
			wantTotal:  1,
			wantGroups: 1,
			wantRules:  []int{0}, // group present, zero rules
		},
		{
			name: "title filter — groups with no matching rules excluded by WHERE",
			req:  &model.SearchAlertRuleRequest{Title: "cpu"},
			prepare: func() {
				mockDB.EXPECT().ExecRaw(gomock.Any(), gomock.Any(), int64(12), "cpu%").
					DoAndReturn(func(out any, _ string, _ ...any) error {
						// SQL WHERE r.title LIKE 'cpu%' naturally eliminates non-matching groups
						rows := []groupRuleRow{
							makeGroupRuleRow(1, "g1", "app", interval, 10, "r1", "g1", "cpu-alert", "ds-1"),
						}
						*(out.(*[]groupRuleRow)) = rows
						return nil
					})
			},
			wantTotal:  1,
			wantGroups: 1,
			wantRules:  []int{1},
		},
		{
			name: "title + groupName combined filter",
			req:  &model.SearchAlertRuleRequest{Title: "cpu", GroupName: "app"},
			prepare: func() {
				mockDB.EXPECT().ExecRaw(gomock.Any(), gomock.Any(), int64(12), "app%", "cpu%").
					DoAndReturn(func(out any, _ string, _ ...any) error {
						rows := []groupRuleRow{
							makeGroupRuleRow(1, "g1", "app", interval, 10, "r1", "g1", "cpu-alert", "ds-1"),
						}
						*(out.(*[]groupRuleRow)) = rows
						return nil
					})
			},
			wantTotal:  1,
			wantGroups: 1,
			wantRules:  []int{1},
		},
		{
			name: "datasourceUID filter",
			req:  &model.SearchAlertRuleRequest{DatasourceUID: "ds-2"},
			prepare: func() {
				mockDB.EXPECT().ExecRaw(gomock.Any(), gomock.Any(), int64(12), "ds-2").
					DoAndReturn(func(out any, _ string, _ ...any) error {
						rows := []groupRuleRow{
							makeGroupRuleRow(3, "g3", "db", interval, 30, "r30", "g3", "pg-alert", "ds-2"),
						}
						*(out.(*[]groupRuleRow)) = rows
						return nil
					})
			},
			wantTotal:  1,
			wantGroups: 1,
			wantRules:  []int{1},
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()

			rs, total, err := srv.SearchAlertRules(ctx, tt.req)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantTotal, total)
			assert.Len(t, rs, tt.wantGroups)
			for i, wantCount := range tt.wantRules {
				assert.Len(t, rs[i].Rules, wantCount, "group[%d] rule count mismatch", i)
			}
		})
	}
}
