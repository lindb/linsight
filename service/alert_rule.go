package service

import (
	"context"
	"strings"

	"github.com/lindb/linsight/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
	"github.com/lindb/linsight/pkg/uuid"
)

type AlertRuleService interface {
	// SearchAlertRules searches the alert rules by given params.
	SearchAlertRules(ctx context.Context, req *model.SearchAlertRuleRequest) (rs []model.AlertRule, total int64, err error)
	// CreateAlertRule creates an alert rule.
	CreateAlertRule(ctx context.Context, rule *model.AlertRule) (string, error)
	// UpdateAlertRule updates the alert rule by uid.
	UpdateAlertRule(ctx context.Context, rule *model.AlertRule) error
	// DeleteAlertRuleByUID deletes the alert rule by uid.
	DeleteAlertRuleByUID(ctx context.Context, uid string) error
	// GetAlertRuleByUID returns the alert rule uid.
	GetAlertRuleByUID(ctx context.Context, uid string) (*model.AlertRule, error)
	FetchAllAlertRules(req *model.SearchAlertRuleRequest) (rs []model.AlertRule, err error)
}

type alertRuleService struct {
	db dbpkg.DB
}

func NewAlertRuleService(db dbpkg.DB) AlertRuleService {
	return &alertRuleService{
		db: db,
	}
}

// SearchAlertRules searches the alert rules by given params.
func (srv *alertRuleService) SearchAlertRules(ctx context.Context, req *model.SearchAlertRuleRequest) (rs []model.AlertRule, total int64, err error) {
	conditions := []string{"org_id=?"}
	signedUser := util.GetUser(ctx)
	params := []any{signedUser.Org.ID}
	if req.Title != "" {
		conditions = append(conditions, "title like ?")
		params = append(params, req.Title+"%")
	}
	where := strings.Join(conditions, " and ")
	count, err := srv.db.Count(&model.AlertRule{}, where, params...)
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
	if err := srv.db.FindForPaging(&rs, offset, limit, "id desc", where, params...); err != nil {
		return nil, 0, err
	}
	return rs, count, nil
}

// CreateAlertRule creates an alert rule.
func (srv *alertRuleService) CreateAlertRule(ctx context.Context, rule *model.AlertRule) (string, error) {
	rule.UID = uuid.GenerateShortUUID()
	// set rule org/user info
	user := util.GetUser(ctx)
	rule.OrgID = user.Org.ID
	userID := user.User.ID
	rule.CreatedBy = userID
	rule.UpdatedBy = userID
	if err := srv.db.Create(rule); err != nil {
		return "", err
	}
	return rule.UID, nil
}

// UpdateAlertRule updates the alert rule by uid.
func (srv *alertRuleService) UpdateAlertRule(ctx context.Context, rule *model.AlertRule) error {
	ruleFromDB, err := srv.GetAlertRuleByUID(ctx, rule.UID)
	if err != nil {
		return err
	}
	user := util.GetUser(ctx)
	// update rule
	ruleFromDB.Title = rule.Title
	ruleFromDB.Data = rule.Data
	ruleFromDB.Conditions = rule.Conditions
	ruleFromDB.Integration = rule.Integration
	ruleFromDB.Notifications = rule.Notifications
	ruleFromDB.Version += 1
	ruleFromDB.UpdatedBy = user.User.ID
	return srv.db.Update(ruleFromDB, "uid=? and org_id=?", rule.UID, user.Org.ID)
}

// DeleteAlertRuleByUID deletes the alert rule by uid.
func (srv *alertRuleService) DeleteAlertRuleByUID(ctx context.Context, uid string) error {
	signedUser := util.GetUser(ctx)
	orgID := signedUser.Org.ID
	return srv.db.Transaction(func(tx dbpkg.DB) error {
		// delete rule
		return tx.Delete(&model.AlertRule{}, "uid=? and org_id=?", uid, orgID)
	})
}

// GetAlertRuleByUID returns the alert rule uid.
func (srv *alertRuleService) GetAlertRuleByUID(ctx context.Context, uid string) (rs *model.AlertRule, err error) {
	signedUser := util.GetUser(ctx)
	if err := srv.db.Get(&rs, "uid=? and org_id=?", uid, signedUser.Org.ID); err != nil {
		return nil, err
	}
	return rs, nil
}

func (srv *alertRuleService) FetchAllAlertRules(req *model.SearchAlertRuleRequest) (rs []model.AlertRule, err error) {
	if err := srv.db.Find(&rs); err != nil {
		return nil, err
	}
	return
}
