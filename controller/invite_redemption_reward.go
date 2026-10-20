package controller

import (
	"math"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
)

const maxInviteRedemptionRewardMultiplier = 100

type inviteRedemptionRewardSettingRequest struct {
	Enabled    bool    `json:"enabled"`
	Multiplier float64 `json:"multiplier"`
}

func GetInviteRedemptionRewardSetting(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    operation_setting.GetInviteRedemptionRewardSetting(),
	})
}

func UpdateInviteRedemptionRewardSetting(c *gin.Context) {
	req := inviteRedemptionRewardSettingRequest{}
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorMsg(c, "无效的参数")
		return
	}
	if req.Enabled && !operation_setting.IsPaymentComplianceConfirmed() {
		common.ApiErrorMsg(c, "请先在支付设置中确认合规声明")
		return
	}
	if req.Enabled && (math.IsNaN(req.Multiplier) || math.IsInf(req.Multiplier, 0) || req.Multiplier <= 0 || req.Multiplier > maxInviteRedemptionRewardMultiplier) {
		common.ApiErrorMsg(c, "奖励倍率必须大于 0 且不超过 100")
		return
	}

	setting := operation_setting.GetInviteRedemptionRewardSetting()
	startTime := setting.StartTime
	if req.Enabled && !setting.Enabled {
		startTime = common.GetTimestamp()
	}
	values := map[string]string{
		"invite_redemption_reward_setting.enabled":    strconv.FormatBool(req.Enabled),
		"invite_redemption_reward_setting.multiplier": strconv.FormatFloat(req.Multiplier, 'f', -1, 64),
		"invite_redemption_reward_setting.start_time": strconv.FormatInt(startTime, 10),
	}
	if err := model.UpdateOptionsBulk(values); err != nil {
		common.ApiErrorMsg(c, "保存邀请兑换奖励配置失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    operation_setting.GetInviteRedemptionRewardSetting(),
	})
}
