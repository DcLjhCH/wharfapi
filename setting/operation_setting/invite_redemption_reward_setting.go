package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

// InviteRedemptionRewardSetting controls rewards earned when a newly invited
// user redeems a balance redemption code. StartTime prevents invitation
// relationships created before the activity began from earning rewards.
type InviteRedemptionRewardSetting struct {
	Enabled    bool    `json:"enabled"`
	Multiplier float64 `json:"multiplier"`
	StartTime  int64   `json:"start_time"`
}

var inviteRedemptionRewardSetting = InviteRedemptionRewardSetting{}

func init() {
	config.GlobalConfig.Register("invite_redemption_reward_setting", &inviteRedemptionRewardSetting)
}

func GetInviteRedemptionRewardSetting() *InviteRedemptionRewardSetting {
	return &inviteRedemptionRewardSetting
}
