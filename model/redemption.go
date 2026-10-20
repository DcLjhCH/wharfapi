package model

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/shopspring/decimal"

	"gorm.io/gorm"
)

type Redemption struct {
	Id           int            `json:"id"`
	UserId       int            `json:"user_id"`
	Key          string         `json:"key" gorm:"type:char(32);uniqueIndex"`
	Status       int            `json:"status" gorm:"default:1"`
	Name         string         `json:"name" gorm:"index"`
	Type         string         `json:"type" gorm:"type:varchar(32);default:'balance'"`
	PlanId       int            `json:"plan_id" gorm:"default:0"`
	Quota        int            `json:"quota" gorm:"default:100"`
	CreatedTime  int64          `json:"created_time" gorm:"bigint"`
	RedeemedTime int64          `json:"redeemed_time" gorm:"bigint"`
	Count        int            `json:"count" gorm:"-:all"` // only for api request
	UsedUserId   int            `json:"used_user_id"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
	ExpiredTime  int64          `json:"expired_time" gorm:"bigint"` // 过期时间，0 表示不过期
}

const (
	RedemptionTypeBalance      = "balance"
	RedemptionTypeSubscription = "subscription"
	RedemptionSource           = "redemption"
)

type RedemptionRedeemResult struct {
	Type           string `json:"type"`
	Quota          int    `json:"quota,omitempty"`
	PlanId         int    `json:"plan_id,omitempty"`
	SubscriptionId int    `json:"subscription_id,omitempty"`
}

func normalizeRedemptionType(redemptionType string) string {
	switch redemptionType {
	case RedemptionTypeSubscription:
		return RedemptionTypeSubscription
	default:
		return RedemptionTypeBalance
	}
}

func GetAllRedemptions(startIdx int, num int) (redemptions []*Redemption, total int64, err error) {
	// 开始事务
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 获取总数
	err = tx.Model(&Redemption{}).Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// 获取分页数据
	err = tx.Order("id desc").Limit(num).Offset(startIdx).Find(&redemptions).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// 提交事务
	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return redemptions, total, nil
}

func SearchRedemptions(keyword string, startIdx int, num int) (redemptions []*Redemption, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Build query based on keyword type
	query := tx.Model(&Redemption{})

	// Only try to convert to ID if the string represents a valid integer
	if id, err := strconv.Atoi(keyword); err == nil {
		query = query.Where("id = ? OR name LIKE ?", id, keyword+"%")
	} else {
		query = query.Where("name LIKE ?", keyword+"%")
	}

	// Get total count
	err = query.Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Get paginated data
	err = query.Order("id desc").Limit(num).Offset(startIdx).Find(&redemptions).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return redemptions, total, nil
}

func GetRedemptionById(id int) (*Redemption, error) {
	if id == 0 {
		return nil, errors.New("id 为空！")
	}
	redemption := Redemption{Id: id}
	var err error = nil
	err = DB.First(&redemption, "id = ?", id).Error
	return &redemption, err
}

func Redeem(key string, userId int) (result *RedemptionRedeemResult, err error) {
	if key == "" {
		return nil, errors.New("未提供兑换码")
	}
	if userId == 0 {
		return nil, errors.New("无效的 user id")
	}
	redemption := &Redemption{}
	result = &RedemptionRedeemResult{}
	var logPlanTitle string
	var upgradeGroup string
	var inviterId int
	var inviteRewardQuota int

	keyCol := "`key`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		keyCol = `"key"`
	}
	common.RandomSleep()
	err = DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Set("gorm:query_option", "FOR UPDATE").Where(keyCol+" = ?", key).First(redemption).Error
		if err != nil {
			return errors.New("无效的兑换码")
		}
		if redemption.Status != common.RedemptionCodeStatusEnabled {
			return errors.New("该兑换码已被使用")
		}
		if redemption.ExpiredTime != 0 && redemption.ExpiredTime < common.GetTimestamp() {
			return errors.New("该兑换码已过期")
		}
		var invitee User
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&invitee, userId).Error; err != nil {
			return err
		}
		redemption.Type = normalizeRedemptionType(redemption.Type)
		result.Type = redemption.Type
		switch redemption.Type {
		case RedemptionTypeSubscription:
			if redemption.PlanId <= 0 {
				return errors.New("subscription redemption plan_id is required")
			}
			plan, err := getSubscriptionPlanByIdTx(tx, redemption.PlanId)
			if err != nil {
				return err
			}
			subscription, err := CreateUserSubscriptionFromPlanTx(tx, userId, plan, RedemptionSource)
			if err != nil {
				return err
			}
			result.PlanId = plan.Id
			result.SubscriptionId = subscription.Id
			logPlanTitle = plan.Title
			upgradeGroup = plan.UpgradeGroup
		default:
			err = tx.Model(&User{}).Where("id = ?", userId).Update("quota", gorm.Expr("quota + ?", redemption.Quota)).Error
			if err != nil {
				return err
			}
			inviterId, inviteRewardQuota, err = grantInviteRedemptionReward(tx, &invitee, redemption.Quota)
			if err != nil {
				return err
			}
			result.Type = RedemptionTypeBalance
			result.Quota = redemption.Quota
		}
		redemption.RedeemedTime = common.GetTimestamp()
		redemption.Status = common.RedemptionCodeStatusUsed
		redemption.UsedUserId = userId
		err = tx.Save(redemption).Error
		return err
	})
	if err != nil {
		common.SysError("redemption failed: " + err.Error())
		return nil, ErrRedeemFailed
	}
	if result.Type == RedemptionTypeSubscription {
		if upgradeGroup != "" {
			_ = UpdateUserGroupCache(userId, upgradeGroup)
		}
		RecordLog(userId, LogTypeTopup, fmt.Sprintf("通过兑换码激活订阅 %s，兑换码ID %d", logPlanTitle, redemption.Id))
		return result, nil
	}
	RecordLog(userId, LogTypeTopup, fmt.Sprintf("通过兑换码充值 %s，兑换码ID %d", logger.LogQuota(redemption.Quota), redemption.Id))
	if inviteRewardQuota > 0 {
		RecordLog(inviterId, LogTypeSystem, fmt.Sprintf("受邀用户兑换码ID %d 成功到账，获得邀请奖励 %s", redemption.Id, logger.LogQuota(inviteRewardQuota)))
	}
	return result, nil
}

// grantInviteRedemptionReward grants an inviter's pending affiliate quota for
// a successful balance-code redemption. It is called inside Redeem's
// transaction, so a redemption code can never create the reward twice.
func grantInviteRedemptionReward(tx *gorm.DB, invitee *User, redeemedQuota int) (int, int, error) {
	setting := operation_setting.GetInviteRedemptionRewardSetting()
	if !setting.Enabled || setting.StartTime <= 0 || setting.Multiplier <= 0 || redeemedQuota <= 0 || invitee == nil {
		return 0, 0, nil
	}
	if invitee.CreatedAt < setting.StartTime || invitee.InviterId <= 0 || invitee.InviterId == invitee.Id {
		return 0, 0, nil
	}

	rewardQuota := decimal.NewFromInt(int64(redeemedQuota)).
		Mul(decimal.NewFromFloat(setting.Multiplier)).
		Floor().IntPart()
	if rewardQuota <= 0 || rewardQuota > int64(^uint(0)>>1) {
		return 0, 0, nil
	}
	result := tx.Model(&User{}).
		Where("id = ?", invitee.InviterId).
		Updates(map[string]interface{}{
			"aff_quota":   gorm.Expr("aff_quota + ?", rewardQuota),
			"aff_history": gorm.Expr("aff_history + ?", rewardQuota),
		})
	if result.Error != nil {
		return 0, 0, result.Error
	}
	if result.RowsAffected != 1 {
		return 0, 0, nil
	}
	return invitee.InviterId, int(rewardQuota), nil
}

func (redemption *Redemption) Insert() error {
	var err error
	err = DB.Create(redemption).Error
	return err
}

func (redemption *Redemption) SelectUpdate() error {
	// This can update zero values
	return DB.Model(redemption).Select("redeemed_time", "status").Updates(redemption).Error
}

// Update Make sure your token's fields is completed, because this will update non-zero values
func (redemption *Redemption) Update() error {
	var err error
	err = DB.Model(redemption).Select("name", "status", "type", "plan_id", "quota", "redeemed_time", "expired_time").Updates(redemption).Error
	return err
}

func (redemption *Redemption) Delete() error {
	var err error
	err = DB.Delete(redemption).Error
	return err
}

func DeleteRedemptionById(id int) (err error) {
	if id == 0 {
		return errors.New("id 为空！")
	}
	redemption := Redemption{Id: id}
	err = DB.Where(redemption).First(&redemption).Error
	if err != nil {
		return err
	}
	return redemption.Delete()
}

func DeleteInvalidRedemptions() (int64, error) {
	now := common.GetTimestamp()
	result := DB.Where("status IN ? OR (status = ? AND expired_time != 0 AND expired_time < ?)", []int{common.RedemptionCodeStatusUsed, common.RedemptionCodeStatusDisabled}, common.RedemptionCodeStatusEnabled, now).Delete(&Redemption{})
	return result.RowsAffected, result.Error
}
