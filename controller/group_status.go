package controller

import (
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

// GetGroupStatus summarizes this site's own recent consume and error logs.
// It does not read request bodies, prompts, or keys, and it does not invent
// prices, quota, or health for groups that had no traffic.
func GetGroupStatus(c *gin.Context) {
	userID := c.GetInt("id")
	role := c.GetInt("role")
	userGroup, err := model.GetUserGroup(userID, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	usable := service.GetUserUsableGroups(userGroup)
	names := service.VisibleGroupNames(role, ratio_setting.GetGroupRatioCopy(), usable)

	candidates := make([]service.GroupStatusCandidate, 0, len(names))
	queryGroups := make([]string, 0, len(names))
	for _, name := range names {
		enabled := model.GetGroupEnabledModels(name)
		if len(enabled) == 0 {
			continue
		}
		candidates = append(candidates, service.GroupStatusCandidate{
			Name:          name,
			Description:   service.GroupStatusDescription(name, usable, setting.GetUsableGroupDescription(name)),
			Ratio:         service.GetUserGroupRatio(userGroup, name),
			EnabledModels: enabled,
		})
		queryGroups = append(queryGroups, name)
	}

	start, end := service.GroupStatusWindow(time.Now().Unix())
	rows, err := model.ListGroupStatusLogs(queryGroups, start, end)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	logs := make([]service.GroupStatusLog, 0, len(rows))
	for _, row := range rows {
		logs = append(logs, service.GroupStatusLog{
			Group:        row.Group,
			Type:         row.Type,
			ModelName:    row.ModelName,
			UseTime:      row.UseTime,
			PromptTokens: row.PromptTokens,
			Other:        row.Other,
			CreatedAt:    row.CreatedAt,
		})
	}

	common.ApiSuccess(c, service.AggregateGroupStatus(start, end, candidates, logs))
}
