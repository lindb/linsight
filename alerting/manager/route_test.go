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

package manager

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"github.com/lindb/linsight/model"
)

// ─── Routing scenario tests ──────────────────────────────────────────────────

func TestTraverseRouteTree_NilRoot(t *testing.T) {
	result := traverseNode(nil, nil, nil)
	assert.Nil(t, result)
}

// No child route matches → root itself is the fallback receiver.
func TestTraverseRouteTree_NoMatch_RootFallback(t *testing.T) {
	root := buildTestTree(t)
	result := traverseNode(root, map[string]string{"alertname": "disk"}, nil)
	require.Len(t, result, 1)
	assert.Equal(t, "default-email", result[0].ContactPointUID)
}

// Both service="database" and env="production" match → deepest node (db-prod) wins.
func TestTraverseRouteTree_DB_Prod(t *testing.T) {
	root := buildTestTree(t)
	result := traverseNode(root, map[string]string{"service": "database", "env": "production"}, nil)
	require.Len(t, result, 1)
	assert.Equal(t, "dba-prod-phone", result[0].ContactPointUID)
}

// Both service="database" and env="testing" match → db-test wins.
func TestTraverseRouteTree_DB_Test(t *testing.T) {
	root := buildTestTree(t)
	result := traverseNode(root, map[string]string{"service": "database", "env": "testing"}, nil)
	require.Len(t, result, 1)
	assert.Equal(t, "dba-test-mutilated", result[0].ContactPointUID)
}

// service="database" matches db-route, but neither child env matches →
// db-route itself is the fallback (parent 兜底).
func TestTraverseRouteTree_DB_NoEnv_ParentFallback(t *testing.T) {
	root := buildTestTree(t)
	result := traverseNode(root, map[string]string{"service": "database"}, nil)
	require.Len(t, result, 1)
	assert.Equal(t, "dba-default-slack", result[0].ContactPointUID)
}

// Only severity="critical" set → db-route not matched, critical-route matched.
func TestTraverseRouteTree_Critical_NonDB(t *testing.T) {
	root := buildTestTree(t)
	result := traverseNode(root, map[string]string{"severity": "critical"}, nil)
	require.Len(t, result, 1)
	assert.Equal(t, "global-oncall-pager", result[0].ContactPointUID)
}

// service="database" AND severity="critical": db-route matches first and
// continue=false, so critical-route (sibling) is never reached.
// The fallback is dba-default-slack (db-route itself, since no env child matched).
func TestTraverseRouteTree_DB_Critical_ContinueFalse(t *testing.T) {
	root := buildTestTree(t)
	result := traverseNode(root, map[string]string{"service": "database", "severity": "critical"}, nil)
	cps := contactPoints(result)
	assert.Contains(t, cps, "dba-default-slack")
	assert.NotContains(t, cps, "global-oncall-pager")
}

// service="database", env="production", severity="critical":
// db-prod (grandchild) is reached; continue=false on db-route still blocks critical-route.
func TestTraverseRouteTree_DB_Prod_Critical_ContinueFalse(t *testing.T) {
	root := buildTestTree(t)
	result := traverseNode(root, map[string]string{
		"service":  "database",
		"env":      "production",
		"severity": "critical",
	}, nil)
	cps := contactPoints(result)
	assert.Contains(t, cps, "dba-prod-phone")
	assert.NotContains(t, cps, "global-oncall-pager")
}

// ─── Inheritance tests ───────────────────────────────────────────────────────

// db-prod has no timing fields set → all values inherited from root.
func TestTraverseRouteTree_Inheritance_TimingFromRoot(t *testing.T) {
	root := buildTestTree(t)
	result := traverseNode(root, map[string]string{"service": "database", "env": "production"}, nil)
	require.Len(t, result, 1)
	r := result[0]
	assert.Equal(t, model.Duration(30*time.Second), r.GroupWait, "GroupWait should be inherited from root")
	assert.Equal(t, model.Duration(5*time.Minute), r.GroupInterval, "GroupInterval should be inherited from root")
	assert.Equal(t, model.Duration(4*time.Hour), r.RepeatInterval, "RepeatInterval should be inherited from root")
	assert.Equal(t, datatypes.JSON(`["alertname"]`), r.GroupBy, "GroupBy should be inherited from root")
}

// db-route overrides GroupWait=10s; all other timing is still zero on db-prod.
// Expected: db-prod inherits GroupWait=10s from db-route (closest ancestor),
// while GroupInterval/RepeatInterval/GroupBy fall through to root.
func TestTraverseRouteTree_Inheritance_PartialOverride(t *testing.T) {
	root := buildTestTree(t)
	root.Routes[0].GroupWait = model.Duration(10 * time.Second)

	result := traverseNode(root, map[string]string{"service": "database", "env": "production"}, nil)
	require.Len(t, result, 1)
	r := result[0]
	assert.Equal(t, model.Duration(10*time.Second), r.GroupWait, "GroupWait should be inherited from db-route (closest ancestor)")
	assert.Equal(t, model.Duration(5*time.Minute), r.GroupInterval, "GroupInterval should fall through to root")
	assert.Equal(t, model.Duration(4*time.Hour), r.RepeatInterval, "RepeatInterval should fall through to root")
	assert.Equal(t, datatypes.JSON(`["alertname"]`), r.GroupBy, "GroupBy should fall through to root")
}
