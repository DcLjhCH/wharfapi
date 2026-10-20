package controller

import (
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
)

const consumeRewardDateLayout = "2006-01-02"

func GetConsumeRewardDailyReport(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	userId, _ := strconv.Atoi(c.Query("user_id"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	if !isConsumeRewardDate(startDate) || !isConsumeRewardDate(endDate) {
		common.ApiErrorMsg(c, "日期格式必须为 YYYY-MM-DD")
		return
	}

	items, total, err := service.GetConsumeRewardDailyReport(service.ConsumeRewardDailyReportFilter{
		UserId:    userId,
		StartDate: startDate,
		EndDate:   endDate,
		Offset:    (page - 1) * pageSize,
		Limit:     pageSize,
	})
	if err != nil {
		common.ApiErrorMsg(c, "查询消费日报失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items, "total": total})
}

func GetConsumeRewardRecords(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	userId, _ := strconv.Atoi(c.Query("user_id"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	records, total, err := service.GetConsumeRewardRecords(userId, (page-1)*pageSize, pageSize)
	if err != nil {
		common.ApiErrorMsg(c, "查询奖励记录失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": records, "total": total})
}

func GetConsumeRewardProgress(c *gin.Context) {
	progress, err := service.GetConsumeRewardProgress(c.GetInt("id"), time.Now())
	if err != nil {
		common.ApiErrorMsg(c, "查询消费奖励进度失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": progress})
}

func GetConsumeRewardSetting(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": operation_setting.GetConsumeRewardSetting()})
}

type consumeRewardSettingRequest struct {
	Enabled         bool `json:"enabled"`
	DailyQuota      int  `json:"daily_quota"`
	RewardQuota     int  `json:"reward_quota"`
	ConsecutiveDays int  `json:"consecutive_days"`
}

func UpdateConsumeRewardSetting(c *gin.Context) {
	req := consumeRewardSettingRequest{}
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorMsg(c, "无效的参数")
		return
	}
	if req.DailyQuota <= 0 || req.RewardQuota <= 0 || req.ConsecutiveDays <= 0 {
		common.ApiErrorMsg(c, "消费门槛、奖励额度和连续天数必须大于 0")
		return
	}
	values := map[string]string{
		"consume_reward_setting.enabled":          strconv.FormatBool(req.Enabled),
		"consume_reward_setting.daily_quota":      strconv.Itoa(req.DailyQuota),
		"consume_reward_setting.reward_quota":     strconv.Itoa(req.RewardQuota),
		"consume_reward_setting.consecutive_days": strconv.Itoa(req.ConsecutiveDays),
	}
	if err := model.UpdateOptionsBulk(values); err != nil {
		common.ApiErrorMsg(c, "保存活动配置失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": operation_setting.GetConsumeRewardSetting()})
}

type consumeRewardManualGrantRequest struct {
	UserId int    `json:"user_id"`
	Quota  int    `json:"quota"`
	Reason string `json:"reason"`
}

func GrantConsumeRewardManually(c *gin.Context) {
	req := consumeRewardManualGrantRequest{}
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorMsg(c, "无效的参数")
		return
	}
	reward, err := service.GrantConsumeRewardManually(req.UserId, req.Quota, c.GetInt("id"), req.Reason)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": reward})
}

type consumeRewardRevokeRequest struct {
	Reason string `json:"reason"`
}

func RevokeConsumeReward(c *gin.Context) {
	recordId, err := strconv.Atoi(c.Param("id"))
	if err != nil || recordId <= 0 {
		common.ApiErrorMsg(c, "无效的奖励记录")
		return
	}
	req := consumeRewardRevokeRequest{}
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorMsg(c, "无效的参数")
		return
	}
	if err := service.RevokeConsumeReward(recordId, c.GetInt("id"), req.Reason); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func isConsumeRewardDate(value string) bool {
	if value == "" {
		return true
	}
	_, err := time.Parse(consumeRewardDateLayout, value)
	return err == nil
}
