package service

import (
	"context"

	"github.com/lindb/linsight/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
)

//go:generate mockgen -source=./dashboard_version.go -destination=./dashboard_version_mock.go -package=service

// DashboardVersionService represents the dashboard version service.
type DashboardVersionService interface {
	// CreateDashboardVersion creates a dashboard version.
	CreateDashboardVersion(ctx context.Context, version *model.DashboardVersion) error
	// ListDashboardVersion returns all dashboard versions based on dashboard uid.
	ListDashboardVersion(ctx context.Context, dashboardUID string) (rs []model.DashboardVersion, err error)
	// GetDashboardVersion returns the dashboard version by dashboard uid and version.
	GetDashboardVersion(ctx context.Context, dashboardUID string, version int) (rs *model.DashboardVersion, err error)
}

type dashboardVersionService struct {
	db dbpkg.DB
}

func NewDashboardVersionService(db dbpkg.DB) DashboardVersionService {
	return &dashboardVersionService{db: db}
}

// CreateDashboardVersion creates a dashboard version.
func (srv *dashboardVersionService) CreateDashboardVersion(ctx context.Context, version *model.DashboardVersion) error {
	return srv.db.Transaction(func(tx dbpkg.DB) error {
		sql := `select COALESCE(max(version),0) from dashboard_versions where dashboard_uid=?`
		var maxVersion int
		if err := srv.db.ExecRaw(&maxVersion, sql, version.DashboardUID); err != nil {
			return err
		}
		user := util.GetUser(ctx)
		userID := user.User.ID
		version.Version = maxVersion + 1
		version.CreatedBy = userID
		version.UpdatedBy = userID
		return tx.Create(version)
	})
}

// ListDashboardVersion returns all dashboard versions based on dashboard uid.
func (srv *dashboardVersionService) ListDashboardVersion(
	ctx context.Context, dashboardUID string,
) (rs []model.DashboardVersion, err error) {
	if err := srv.db.ExecRaw(&rs,
		`select version,dashboard_uid,message,created_at from dashboard_versions where dashboard_uid=? order by version desc`,
		dashboardUID); err != nil {
		return nil, err
	}
	return
}

// GetDashboardVersion returns the dashboard version by dashboard uid and version.
func (srv *dashboardVersionService) GetDashboardVersion(
	ctx context.Context, dashboardUID string, version int,
) (rs *model.DashboardVersion, err error) {
	if err := srv.db.Get(&rs, "dashboard_uid=? and version=?",
		dashboardUID, version); err != nil {
		return nil, err
	}
	return
}
