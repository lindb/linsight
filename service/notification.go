package service

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/lindb/linsight/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
	"github.com/lindb/linsight/pkg/uuid"
)

type NotificationService interface {
	GetAllNotificationSettings(ctx context.Context) (rs []model.NotificationSetting, err error)
	SaveNotificationSetting(ctx context.Context, setting *model.NotificationSetting) error
	DeleteNotificationSetting(ctx context.Context, notificationType string) error
	GetNotificationSettingsByOrg(orgID int64) (rs []model.NotificationSetting, err error)

	CreateNotification(ctx context.Context, notification *model.Notification) (string, error)
	UpdateNotification(ctx context.Context, notification *model.Notification) error
	GetNotificationByUID(ctx context.Context, uid string) (*model.Notification, error)
	FindNotifications(ctx context.Context, req *model.SearchNotificationRequest) (rs []model.Notification, total int64, err error)
	GetNotificationByOrgAndUID(orgID int64, uid string) (*model.Notification, error)
}

type notificationService struct {
	db dbpkg.DB
}

func NewNotificationService(db dbpkg.DB) NotificationService {
	return &notificationService{
		db: db,
	}
}

func (srv *notificationService) GetAllNotificationSettings(ctx context.Context) (rs []model.NotificationSetting, err error) {
	user := util.GetUser(ctx)
	return srv.GetNotificationSettingsByOrg(user.Org.ID)
}

func (srv *notificationService) GetNotificationSettingsByOrg(orgID int64) (rs []model.NotificationSetting, err error) {
	if err := srv.db.Find(&rs, "org_id=?", orgID); err != nil {
		return nil, err
	}
	return
}

func (srv *notificationService) SaveNotificationSetting(ctx context.Context, setting *model.NotificationSetting) error {
	user := util.GetUser(ctx)
	return srv.db.Transaction(func(tx dbpkg.DB) error {
		var settingFromDB model.NotificationSetting
		err := tx.Get(&settingFromDB, "org_id=? and type=?", user.Org.ID, setting.Type)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			setting.OrgID = user.Org.ID
			return tx.Create(&setting)
		}
		if err != nil {
			return err
		}
		settingFromDB.Config = setting.Config
		return tx.Update(settingFromDB, "org_id=? and type=?", user.Org.ID, setting.Type)
	})
}

func (srv *notificationService) DeleteNotificationSetting(ctx context.Context, notificationType string) error {
	user := util.GetUser(ctx)
	return srv.db.Delete(&model.NotificationSetting{}, "org_id=? and type=?", user.Org.ID, notificationType)
}

func (srv *notificationService) CreateNotification(ctx context.Context, notification *model.Notification) (string, error) {
	notification.UID = uuid.GenerateShortUUID()
	// set notification org/user info
	user := util.GetUser(ctx)
	notification.OrgID = user.Org.ID
	userID := user.User.ID
	notification.CreatedBy = userID
	notification.UpdatedBy = userID
	if err := srv.db.Create(notification); err != nil {
		return "", err
	}
	return notification.UID, nil
}

func (srv *notificationService) UpdateNotification(ctx context.Context, notification *model.Notification) error {
	notificationFromDB, err := srv.GetNotificationByUID(ctx, notification.UID)
	if err != nil {
		return err
	}
	user := util.GetUser(ctx)
	// update notification
	notificationFromDB.Name = notification.Name
	notificationFromDB.Receviers = notification.Receviers
	notificationFromDB.UpdatedBy = user.User.ID
	return srv.db.Update(notificationFromDB, "uid=? and org_id=?", notification.UID, user.Org.ID)
}

func (srv *notificationService) FindNotifications(ctx context.Context, req *model.SearchNotificationRequest) (rs []model.Notification, total int64, err error) {
	conditions := []string{"org_id=?"}
	signedUser := util.GetUser(ctx)
	params := []any{signedUser.Org.ID}
	if req.Name != "" {
		conditions = append(conditions, "name like ?")
		params = append(params, req.Name+"%")
	}
	where := strings.Join(conditions, " and ")
	count, err := srv.db.Count(&model.Notification{}, where, params...)
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

func (srv *notificationService) GetNotificationByUID(ctx context.Context, uid string) (*model.Notification, error) {
	signedUser := util.GetUser(ctx)
	return srv.GetNotificationByOrgAndUID(signedUser.Org.ID, uid)
}

func (srv *notificationService) GetNotificationByOrgAndUID(orgID int64, uid string) (rs *model.Notification, err error) {
	if err := srv.db.Get(&rs, "uid=? and org_id=?", uid, orgID); err != nil {
		return nil, err
	}
	return rs, nil
}
