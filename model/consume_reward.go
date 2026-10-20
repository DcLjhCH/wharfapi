package model

// ConsumeRewardDaily stores one user's net eligible consumption for a calendar
// day. Amounts are quota integers, never floating point currency values.
type ConsumeRewardDaily struct {
	Id           int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId       int    `json:"user_id" gorm:"not null;uniqueIndex:idx_consume_reward_daily_user_date;index"`
	ConsumeDate  string `json:"consume_date" gorm:"type:varchar(10);not null;uniqueIndex:idx_consume_reward_daily_user_date;index"`
	ConsumeQuota int    `json:"consume_quota" gorm:"not null;default:0"`
	RefundQuota  int    `json:"refund_quota" gorm:"not null;default:0"`
	NetQuota     int    `json:"net_quota" gorm:"not null;default:0"`
	Qualified    bool   `json:"qualified" gorm:"not null;default:false;index"`
	CreatedAt    int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt    int64  `json:"updated_at" gorm:"bigint"`
}

func (ConsumeRewardDaily) TableName() string {
	return "consume_reward_dailies"
}

// ConsumeRewardRecord is the immutable ledger of rewards that were granted or
// later revoked. RewardKey makes automatic grants idempotent across retries.
type ConsumeRewardRecord struct {
	Id          int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId      int    `json:"user_id" gorm:"not null;uniqueIndex:idx_consume_reward_user_key,priority:1;index"`
	RewardKey   string `json:"reward_key" gorm:"type:varchar(64);not null;uniqueIndex:idx_consume_reward_user_key,priority:2"`
	Cycle       int    `json:"cycle" gorm:"not null;default:0"`
	RewardDate  string `json:"reward_date" gorm:"type:varchar(10);not null;index"`
	RewardQuota int    `json:"reward_quota" gorm:"not null"`
	Source      string `json:"source" gorm:"type:varchar(16);not null"`
	Status      string `json:"status" gorm:"type:varchar(16);not null;index"`
	OperatorId  int    `json:"operator_id" gorm:"not null;default:0"`
	Reason      string `json:"reason" gorm:"type:varchar(255);not null;default:''"`
	CreatedAt   int64  `json:"created_at" gorm:"bigint"`
	RevokedAt   int64  `json:"revoked_at" gorm:"bigint;default:0"`
}

func (ConsumeRewardRecord) TableName() string {
	return "consume_reward_records"
}

