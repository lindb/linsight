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

	"gorm.io/gorm"

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
	// CreateAlertRoute creates a route node. When req.RefUID is set, the new node
	// is inserted as a sibling before or after the referenced node.
	CreateAlertRoute(ctx context.Context, req *model.CreateAlertRouteRequest) (string, error)
	// UpdateAlertRoute updates a route node.
	UpdateAlertRoute(ctx context.Context, route *model.AlertRoute) error
	// DeleteAlertRouteByUID removes a route node and — for non-default nodes —
	// cascades the deletion to the entire subtree rooted at that node.
	DeleteAlertRouteByUID(ctx context.Context, uid string) error
	// GetAlertRouteByUID returns a single route node.
	GetAlertRouteByUID(ctx context.Context, uid string) (*model.AlertRoute, error)
	// SearchAlertRoutes searches routes by name.
	SearchAlertRoutes(ctx context.Context, req *model.SearchAlertRouteRequest) (rs []model.AlertRoute, total int64, err error)
	// SortAlertRoutes reorders sibling routes by assigning sequential sort_order values.
	SortAlertRoutes(ctx context.Context, routes []model.AlertRoute) error
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

func (srv *alertRouteService) CreateAlertRoute(ctx context.Context, req *model.CreateAlertRouteRequest) (string, error) {
	user := util.GetUser(ctx)
	orgID := user.Org.ID
	route := &req.AlertRoute

	// Root route (ParentUID="") is created automatically when an org is initialized.
	// Callers must always specify a parent; creating a root route via API is not allowed.
	if route.ParentUID == "" {
		return "", fmt.Errorf("parentUID is required; the root route is created automatically when the org is initialized")
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
	exists, err := srv.db.Exist(&model.AlertRoute{}, "org_id=? and uid=?", orgID, route.ParentUID)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", fmt.Errorf("parent route %q not found", route.ParentUID)
	}

	route.UID = uuid.GenerateShortUUID()
	route.OrgID = orgID
	route.CreatedBy = user.User.ID
	route.UpdatedBy = user.User.ID

	if req.RefUID == "" {
		// Simple create: no sibling positioning requested.
		if err := srv.db.Create(route); err != nil {
			return "", err
		}
		return route.UID, nil
	}

	// Sibling insertion: resolve the reference node inside the transaction to get
	// a consistent snapshot, validate it is a true sibling, then shift existing
	// siblings to make room and insert — all atomically.
	if err := srv.db.Transaction(func(tx dbpkg.DB) error {
		var refRoute model.AlertRoute
		if err := tx.Get(&refRoute, "uid=? and org_id=?", req.RefUID, orgID); err != nil {
			return fmt.Errorf("reference route %q not found", req.RefUID)
		}
		if refRoute.ParentUID != route.ParentUID {
			return fmt.Errorf("reference route %q is not a sibling: expected parentUID %q, got %q",
				req.RefUID, route.ParentUID, refRoute.ParentUID)
		}
		targetOrder := refRoute.SortOrder
		if req.Position == "after" {
			targetOrder = refRoute.SortOrder + 1
		}
		route.SortOrder = targetOrder
		// Shift siblings at or after the target position to create a gap.
		if err := tx.RawDB().Model(&model.AlertRoute{}).
			Where("org_id=? and parent_uid=? and sort_order>=?", orgID, route.ParentUID, targetOrder).
			UpdateColumn("sort_order", gorm.Expr("sort_order + 1")).Error; err != nil {
			return err
		}
		return tx.Create(route)
	}); err != nil {
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

	// If caller tries to re-parent the root (ParentUID=""), reject it to preserve the tree invariant.
	if fromDB.ParentUID == "" && route.ParentUID != "" {
		return fmt.Errorf("cannot re-parent the root route")
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

	if route.ParentUID == "" {
		// Refuse to delete the root route while it still has child nodes to prevent
		// accidentally orphaning the entire policy tree.
		count, err := srv.db.Count(&model.AlertRoute{}, "org_id=? and parent_uid=?", orgID, uid)
		if err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("cannot delete the root route while it has %d child node(s); delete children first", count)
		}
		return srv.db.Delete(&model.AlertRoute{}, "uid=? and org_id=?", uid, orgID)
	}

	// For non-root routes, BFS-collect the entire subtree, then delete all in one transaction.
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

	// Index children by parent UID; root is identified by empty ParentUID.
	childrenMap := make(map[string][]*model.AlertRoute, len(routes))
	var root *model.AlertRoute
	for _, r := range routes {
		if r.ParentUID == "" {
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
		node.Routes = append(node.Routes, buildNode(child, childrenMap))
	}
	return node
}

func (srv *alertRouteService) SortAlertRoutes(ctx context.Context, routes []model.AlertRoute) error {
	user := util.GetUser(ctx)
	return srv.db.Transaction(func(tx dbpkg.DB) error {
		for i, route := range routes {
			if err := tx.Updates(&model.AlertRoute{}, map[string]any{
				"sort_order": i,
				"updated_by": user.User.ID,
			}, "uid=? and org_id=?", route.UID, user.Org.ID); err != nil {
				return err
			}
		}
		return nil
	})
}
