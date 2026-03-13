package api

import (
	"strconv"

	"github.com/gin-gonic/gin"
	httppkg "github.com/lindb/common/pkg/http"

	"github.com/lindb/linsight/constant"
	apideps "github.com/lindb/linsight/http/deps"
)

// DashboardVersionAPI represents dashboard version related api handlers.
type DashboardVersionAPI struct {
	deps apideps.API
}

// NewDashboardVersionAPI creates a DashboardVersionAPI instance.
func NewDashboardVersionAPI(deps *apideps.API) *DashboardVersionAPI {
	return &DashboardVersionAPI{
		deps: *deps,
	}
}

// GetDashboardVersions returns all dashboard versions.
func (api *DashboardVersionAPI) GetDashboardVersions(c *gin.Context) {
	uid := c.Param(constant.UID)
	ds, err := api.deps.DashboardVersionSrv.ListDashboardVersion(c.Request.Context(), uid)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, ds)
}

// GetDashboardVersion returns dashboard version by uid and version.
func (api *DashboardVersionAPI) GetDashboardVersion(c *gin.Context) {
	uid := c.Param(constant.UID)
	version := c.Param("version")
	v, _ := strconv.ParseInt(version, 10, 32)
	ds, err := api.deps.DashboardVersionSrv.GetDashboardVersion(c.Request.Context(), uid, int(v))
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, ds)
}
