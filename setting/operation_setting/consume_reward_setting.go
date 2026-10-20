package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

// ConsumeRewardSetting controls the recurring consecutive-consumption reward.
// All amounts use internal quota units rather than floating point currency.
type ConsumeRewardSetting struct {
	Enabled         bool `json:"enabled"`
	DailyQuota      int  `json:"daily_quota"`
	RewardQuota     int  `json:"reward_quota"`
	ConsecutiveDays int  `json:"consecutive_days"`
}

var consumeRewardSetting = ConsumeRewardSetting{
	Enabled:         false,
	DailyQuota:      0,
	RewardQuota:     0,
	ConsecutiveDays: 7,
}

func init() {
	config.GlobalConfig.Register("consume_reward_setting", &consumeRewardSetting)
}

func GetConsumeRewardSetting() *ConsumeRewardSetting {
	return &consumeRewardSetting
}

