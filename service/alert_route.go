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
	"fmt"
	"sort"
	"strings"

	"github.com/lindb/linsight/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
	"github.com/lindb/linsight/pkg/uuid"
)

// AlertRouteService manages alert routing tree nodes.
type AlertRouteService interface {
	// GetAlertRouteTree returns the routing tree root for the current org.
	// Returns nil when no default (root) route has been created yet.
	GetAlertRouteTree(ctx context.Context) (*model.AlertRouteNode, error)
	// GetAlertRouteTreeByOrg returns the routing tree root for the given org.
	// This variant is used by the alert engine which has no signed-in user context.
	GetAlertRouteTreeByOrg(orgID int64) (*model.AlertRouteNode, error)
	// CreateAlertRoute creates a route node.
	CreateAlertRoute(ctx context.Context, route *model.AlertRoute) (string, error)
	// UpdateAlertRoute updates a route node.
	UpdateAlertRoute(ctx context.Context, route *model.AlertRoute) error
	// DeleteAlertRouteByUID removes a route node and — for non-default nodes —
	// cascades the deletion to the entire subtree rooted at that node.
	DeleteAlertRouteByUID(ctx context.Context, uid string) error
	// GetAlertRouteByUID returns a single route node.
	GetAlertRouteByUID(ctx context.Context, uid string) (*model.AlertRoute, error)
	// SearchAlertRoutes searches routes by name.
	SearchAlertRoutes(ctx context.Context, req *model.SearchAlertRouteRequest) (rs []model.AlertRoute, total int64, err error)
}

type alertRouteService struct {
	db dbpkg.DB
}

// NewAlertRouteService creates an AlertRouteService instance.
func NewAlertRouteService(db dbpkg.DB) AlertRouteService {
	return &alertRouteService{db: db}
}

func (srv *alertRouteService) GetAlertRouteTree(ctx context.Context) (*model.AlertRouteNode, error) {
	signedUser := util.GetUser(ctx)
	return srv.GetAlertRouteTreeByOrg(signedUser.Org.ID)
}

func (srv *alertRouteService) GetAlertRouteTreeByOrg(orgID int64) (*model.AlertRouteNode, error) {
	routes, err := srv.getAlertRoutesByOrg(orgID)
	if err != nil {
		return nil, err
	}
	return buildRouteTree(routes), nil
}

func (srv *alertRouteService) CreateAlertRoute(ctx context.Context, route *model.AlertRoute) (string, error) {
	user := util.GetUser(ctx)
	orgID := user.Org.ID

	// Enforce at-most-one default route per org
	if route.IsDefault {
		count, err := srv.db.Count(&model.AlertRoute{}, "org_id=? and is_default=?", orgID, true)
		if err != nil {
			return "", err
		}
		if count > 0 {
			return "", fmt.Errorf("org already has a default (root) alert route; only one is allowed")
		}
	}

	// Validate that the referenced contact point exists
	if route.ContactPointUID != "" {
		exists, err := srv.db.Exist(&model.ContactPoint{}, "org_id=? and uid=?", orgID, route.ContactPointUID)
		if err != nil {
			return "", err
		}
		if !exists {
			return "", fmt.Errorf("contact point %q not found", route.ContactPointUID)
		}
	}

	// Validate that the referenced parent route exists
	if route.ParentUID != "" {
		exists, err := srv.db.Exist(&model.AlertRoute{}, "org_id=? and uid=?", orgID, route.ParentUID)
		if err != nil {
			return "", err
		}
		if !exists {
			return "", fmt.Errorf("parent route %q not found", route.ParentUID)
		}
	}

	route.UID = uuid.GenerateShortUUID()
	route.OrgID = orgID
	route.CreatedBy = user.User.ID
	route.UpdatedBy = user.User.ID
	if err := srv.db.Create(route); err != nil {
		return "", err
	}
	return route.UID, nil
}

func (srv *alertRouteService) UpdateAlertRoute(ctx context.Context, route *model.AlertRoute) error {
	fromDB, err := srv.GetAlertRouteByUID(ctx, route.UID)
	if err != nil {
		return err
	}
	user := util.GetUser(ctx)
	orgID := user.Org.ID

	// If caller is promoting this route to default, ensure no other default exists
	if route.IsDefault && !fromDB.IsDefault {
		count, err := srv.db.Count(&model.AlertRoute{}, "org_id=? and is_default=?", orgID, true)
		if err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("org already has a default (root) alert route; only one is allowed")
		}
	}

	// Validate that the referenced contact point exists
	if route.ContactPointUID != "" {
		exists, err := srv.db.Exist(&model.ContactPoint{}, "org_id=? and uid=?", orgID, route.ContactPointUID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("contact point %q not found", route.ContactPointUID)
		}
	}

	// Validate that the referenced parent route exists (cannot re-parent to itself)
	if route.ParentUID != "" && route.ParentUID != fromDB.ParentUID {
		if route.ParentUID == route.UID {
			return fmt.Errorf("a route cannot be its own parent")
		}
		exists, err := srv.db.Exist(&model.AlertRoute{}, "org_id=? and uid=?", orgID, route.ParentUID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("parent route %q not found", route.ParentUID)
		}
	}

	fromDB.Name = route.Name
	fromDB.IsDefault = route.IsDefault
	fromDB.ParentUID = route.ParentUID
	fromDB.Matchers = route.Matchers
	fromDB.ContactPointUID = route.ContactPointUID
	fromDB.Continue = route.Continue
	fromDB.GroupBy = route.GroupBy
	fromDB.GroupWait = route.GroupWait
	fromDB.GroupInterval = route.GroupInterval
	fromDB.RepeatInterval = route.RepeatInterval
	fromDB.SortOrder = route.SortOrder
	fromDB.UpdatedBy = user.User.ID
	return srv.db.Update(fromDB, "uid=? and org_id=?", route.UID, orgID)
}

func (srv *alertRouteService) DeleteAlertRouteByUID(ctx context.Context, uid string) error {
	signedUser := util.GetUser(ctx)
	orgID := signedUser.Org.ID

	route, err := srv.GetAlertRouteByUID(ctx, uid)
	if err != nil {
		return err
	}

	if route.IsDefault {
		// Refuse to delete the root route while it still has child nodes to prevent
		// accidentally orphaning the entire policy tree (mirrors Grafana behaviour).
		count, err := srv.db.Count(&model.AlertRoute{}, "org_id=? and parent_uid=?", orgID, uid)
		if err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("cannot delete the default (root) route while it has %d child node(s); delete children first", count)
		}
		return srv.db.Delete(&model.AlertRoute{}, "uid=? and org_id=?", uid, orgID)
	}

	// For non-default routes, BFS-collect the entire subtree, then delete all in one transaction.
	toDelete := []string{uid}
	queue := []string{uid}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		var children []*model.AlertRoute
		if err := srv.db.Find(&children, "org_id=? and parent_uid=?", orgID, current); err != nil {
			return err
		}
		for _, child := range children {
			toDelete = append(toDelete, child.UID)
			queue = append(queue, child.UID)
		}
	}

	return srv.db.Transaction(func(tx dbpkg.DB) error {
		for _, deleteUID := range toDelete {
			if err := tx.Delete(&model.AlertRoute{}, "uid=? and org_id=?", deleteUID, orgID); err != nil {
				return err
			}
		}
		return nil
	})
}

func (srv *alertRouteService) GetAlertRouteByUID(ctx context.Context, uid string) (rs *model.AlertRoute, err error) {
	signedUser := util.GetUser(ctx)
	if err := srv.db.Get(&rs, "uid=? and org_id=?", uid, signedUser.Org.ID); err != nil {
		return nil, err
	}
	return rs, nil
}

func (srv *alertRouteService) SearchAlertRoutes(ctx context.Context, req *model.SearchAlertRouteRequest) (rs []model.AlertRoute, total int64, err error) {
	signedUser := util.GetUser(ctx)
	conditions := []string{"org_id=?"}
	params := []any{signedUser.Org.ID}
	if req.Name != "" {
		conditions = append(conditions, "name like ?")
		params = append(params, req.Name+"%")
	}
	where := strings.Join(conditions, " and ")
	count, err := srv.db.Count(&model.AlertRoute{}, where, params...)
	if err != nil {
		return nil, 0, err
	}
	if count == 0 {
		return nil, 0, nil
	}
	offset := 0
	limit := 20
	if req.Offset > 0 {
		offset = req.Offset
	}
	if req.Limit > 0 {
		limit = req.Limit
	}
	if err := srv.db.FindForPaging(&rs, offset, limit, "sort_order asc, id asc", where, params...); err != nil {
		return nil, 0, err
	}
	return rs, count, nil
}

// getAlertRoutesByOrg returns all routes for an org, ordered by sort_order then id.
// This is an internal helper used to build the routing tree.
func (srv *alertRouteService) getAlertRoutesByOrg(orgID int64) ([]*model.AlertRoute, error) {
	var rs []*model.AlertRoute
	if err := srv.db.Find(&rs, "org_id=?", orgID); err != nil {
		return nil, err
	}
	return rs, nil
}

// buildRouteTree converts a flat list of AlertRoutes into a nested tree rooted at the
// default route.  Returns nil when no default route is present.
func buildRouteTree(routes []*model.AlertRoute) *model.AlertRouteNode {
	if len(routes) == 0 {
		return nil
	}

	// Index children by parent UID
	childrenMap := make(map[string][]*model.AlertRoute, len(routes))
	var root *model.AlertRoute
	for _, r := range routes {
		if r.IsDefault {
			root = r
		} else {
			childrenMap[r.ParentUID] = append(childrenMap[r.ParentUID], r)
		}
	}
	if root == nil {
		return nil
	}

	// Sort each child list by sort_order asc, then id asc for stable ordering
	for parentUID := range childrenMap {
		sort.Slice(childrenMap[parentUID], func(i, j int) bool {
			a, b := childrenMap[parentUID][i], childrenMap[parentUID][j]
			if a.SortOrder != b.SortOrder {
				return a.SortOrder < b.SortOrder
			}
			return a.ID < b.ID
		})
	}

	return buildNode(root, childrenMap)
}

// buildNode recursively constructs an AlertRouteNode for the given route.
func buildNode(route *model.AlertRoute, childrenMap map[string][]*model.AlertRoute) *model.AlertRouteNode {
	node := &model.AlertRouteNode{AlertRoute: *route}
	for _, child := range childrenMap[route.UID] {
		node.Children = append(node.Children, buildNode(child, childrenMap))
	}
	return node
}
