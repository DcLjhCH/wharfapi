package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	consumeRewardDateLayout      = "2006-01-02"
	consumeRewardSourceAutomatic = "automatic"
	consumeRewardSourceManual    = "manual"
	consumeRewardStatusGranted   = "granted"
	consumeRewardStatusRevoked   = "revoked"
)

var consumeRewardLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

type ConsumeRewardDailyReportFilter struct {
	UserId    int
	StartDate string
	EndDate   string
	Offset    int
	Limit     int
}

type ConsumeRewardDailyReportItem struct {
	UserId       int    `json:"user_id"`
	Username     string `json:"username"`
	ConsumeDate  string `json:"consume_date"`
	ConsumeQuota int    `json:"consume_quota"`
	RefundQuota  int    `json:"refund_quota"`
	NetQuota     int    `json:"net_quota"`
	Qualified    bool   `json:"qualified"`
}

// ConsumeRewardProgress is the user-visible view of one user's recurring
// consumption reward activity. It intentionally contains no cross-user data.
type ConsumeRewardProgress struct {
	Enabled              bool                          `json:"enabled"`
	DailyQuota           int                           `json:"daily_quota"`
	RewardQuota          int                           `json:"reward_quota"`
	ConsecutiveDays      int                           `json:"consecutive_days"`
	Today                string                        `json:"today"`
	TodayNetQuota        int                           `json:"today_net_quota"`
	TodayQualified       bool                          `json:"today_qualified"`
	CurrentStreak        int                           `json:"current_streak"`
	RemainingDays        int                           `json:"remaining_days"`
	CalendarMonth        string                        `json:"calendar_month"`
	CalendarStartWeekday int                           `json:"calendar_start_weekday"`
	CalendarDays         []ConsumeRewardProgressDay    `json:"calendar_days"`
	Rewards              []ConsumeRewardProgressReward `json:"rewards"`
}

type ConsumeRewardProgressDay struct {
	ConsumeDate string `json:"consume_date"`
	NetQuota    int    `json:"net_quota"`
	Qualified   bool   `json:"qualified"`
}

type ConsumeRewardProgressReward struct {
	Id          int    `json:"id"`
	RewardDate  string `json:"reward_date"`
	RewardQuota int    `json:"reward_quota"`
	Source      string `json:"source"`
	Status      string `json:"status"`
}

func GetConsumeRewardDailyReport(filter ConsumeRewardDailyReportFilter) ([]ConsumeRewardDailyReportItem, int64, error) {
	query := model.DB.Table("consume_reward_dailies AS daily").
		Select("daily.user_id, users.username, daily.consume_date, daily.consume_quota, daily.refund_quota, daily.net_quota, daily.qualified").
		Joins("LEFT JOIN users ON users.id = daily.user_id")
	if filter.UserId > 0 {
		query = query.Where("daily.user_id = ?", filter.UserId)
	}
	if filter.StartDate != "" {
		query = query.Where("daily.consume_date >= ?", filter.StartDate)
	}
	if filter.EndDate != "" {
		query = query.Where("daily.consume_date <= ?", filter.EndDate)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []ConsumeRewardDailyReportItem
	if err := query.Order("daily.consume_date DESC, daily.user_id ASC").Limit(filter.Limit).Offset(filter.Offset).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func GetConsumeRewardRecords(userId int, offset int, limit int) ([]model.ConsumeRewardRecord, int64, error) {
	query := model.DB.Model(&model.ConsumeRewardRecord{})
	if userId > 0 {
		query = query.Where("user_id = ?", userId)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var records []model.ConsumeRewardRecord
	if err := query.Order("reward_date DESC, id DESC").Limit(limit).Offset(offset).Find(&records).Error; err != nil {
		return nil, 0, err
	}
	return records, total, nil
}

func GetConsumeRewardProgress(userId int, now time.Time) (*ConsumeRewardProgress, error) {
	setting := operation_setting.GetConsumeRewardSetting()
	today := now.In(consumeRewardLocation)
	todayDate := today.Format(consumeRewardDateLayout)
	progress := &ConsumeRewardProgress{
		Enabled:              setting.Enabled,
		DailyQuota:           setting.DailyQuota,
		RewardQuota:          setting.RewardQuota,
		ConsecutiveDays:      setting.ConsecutiveDays,
		Today:                todayDate,
		CalendarMonth:        today.Format("2006-01"),
		CalendarStartWeekday: (int(time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, consumeRewardLocation).Weekday()) + 6) % 7,
		CalendarDays:         make([]ConsumeRewardProgressDay, 0, 31),
		Rewards:              make([]ConsumeRewardProgressReward, 0),
	}
	if !setting.Enabled || setting.DailyQuota <= 0 || setting.RewardQuota <= 0 || setting.ConsecutiveDays <= 0 {
		return progress, nil
	}

	monthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, consumeRewardLocation)
	monthEnd := monthStart.AddDate(0, 1, 0).AddDate(0, 0, -1)
	startDate := monthStart.Format(consumeRewardDateLayout)
	endDate := monthEnd.Format(consumeRewardDateLayout)
	var dailies []model.ConsumeRewardDaily
	if err := model.DB.Where("user_id = ? AND consume_date >= ? AND consume_date <= ?", userId, startDate, endDate).
		Find(&dailies).Error; err != nil {
		return nil, err
	}
	dailyByDate := make(map[string]model.ConsumeRewardDaily, len(dailies))
	for _, daily := range dailies {
		dailyByDate[daily.ConsumeDate] = daily
	}
	for dateCursor := monthStart; !dateCursor.After(monthEnd); dateCursor = dateCursor.AddDate(0, 0, 1) {
		date := dateCursor.Format(consumeRewardDateLayout)
		daily := dailyByDate[date]
		progress.CalendarDays = append(progress.CalendarDays, ConsumeRewardProgressDay{
			ConsumeDate: date,
			NetQuota:    daily.NetQuota,
			Qualified:   daily.Qualified,
		})
	}
	if daily, ok := dailyByDate[todayDate]; ok {
		progress.TodayNetQuota = daily.NetQuota
		progress.TodayQualified = daily.Qualified
	}

	lastAwardDate, err := lastAutomaticRewardDate(model.DB, userId)
	if err != nil {
		return nil, err
	}

	streak, err := consecutiveQualifiedDays(model.DB, userId, lastAwardDate, todayDate)
	if err != nil {
		return nil, err
	}
	progress.CurrentStreak = streak
	progress.RemainingDays = max(setting.ConsecutiveDays-streak, 0)

	var rewards []model.ConsumeRewardRecord
	if err = model.DB.Where("user_id = ?", userId).Order("reward_date DESC, id DESC").Limit(20).Find(&rewards).Error; err != nil {
		return nil, err
	}
	for _, reward := range rewards {
		progress.Rewards = append(progress.Rewards, ConsumeRewardProgressReward{
			Id:          reward.Id,
			RewardDate:  reward.RewardDate,
			RewardQuota: reward.RewardQuota,
			Source:      reward.Source,
			Status:      reward.Status,
		})
	}
	return progress, nil
}

func GrantConsumeRewardManually(userId int, quota int, operatorId int, reason string) (*model.ConsumeRewardRecord, error) {
	if userId <= 0 || quota <= 0 {
		return nil, errors.New("user and quota must be positive")
	}
	reward := &model.ConsumeRewardRecord{
		UserId: userId, RewardKey: "manual:" + common.NewRequestId(), RewardDate: time.Now().In(consumeRewardLocation).Format("2006-01-02"),
		RewardQuota: quota, Source: consumeRewardSourceManual, Status: consumeRewardStatusGranted, OperatorId: operatorId, Reason: reason, CreatedAt: common.GetTimestamp(),
	}
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(reward).Error; err != nil {
			return err
		}
		return tx.Model(&model.User{}).Where("id = ?", userId).UpdateColumn("quota", gorm.Expr("quota + ?", quota)).Error
	})
	if err != nil {
		return nil, err
	}
	_ = model.InvalidateUserCache(userId)
	model.RecordLog(userId, model.LogTypeSystem, fmt.Sprintf("管理员补发连续消费奖励，获得额度 %s", logger.LogQuota(quota)))
	return reward, nil
}

func RevokeConsumeReward(recordId int, operatorId int, reason string) error {
	if recordId <= 0 || reason == "" {
		return errors.New("record and reason are required")
	}
	var reward model.ConsumeRewardRecord
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", recordId).First(&reward).Error; err != nil {
			return err
		}
		if reward.Status != consumeRewardStatusGranted {
			return errors.New("reward is not active")
		}
		result := tx.Model(&model.User{}).Where("id = ? AND quota >= ?", reward.UserId, reward.RewardQuota).
			UpdateColumn("quota", gorm.Expr("quota - ?", reward.RewardQuota))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("user balance is insufficient to revoke this reward")
		}
		return tx.Model(&model.ConsumeRewardRecord{}).Where("id = ?", reward.Id).Updates(map[string]interface{}{
			"status": consumeRewardStatusRevoked, "operator_id": operatorId, "reason": reason, "revoked_at": common.GetTimestamp(),
		}).Error
	})
	if err != nil {
		return err
	}
	_ = model.InvalidateUserCache(reward.UserId)
	model.RecordLog(reward.UserId, model.LogTypeSystem, fmt.Sprintf("管理员撤销连续消费奖励，扣回额度 %s", logger.LogQuota(reward.RewardQuota)))
	return nil
}

// RecordConsumeRewardConsumption records settled, positive consumption for the
// recurring reward. It returns a reward only when this event completes a new
// qualifying streak.
func RecordConsumeRewardConsumption(userId int, quota int, occurredAt time.Time) (*model.ConsumeRewardRecord, error) {
	setting := operation_setting.GetConsumeRewardSetting()
	if !setting.Enabled || quota <= 0 || setting.DailyQuota <= 0 || setting.RewardQuota <= 0 || setting.ConsecutiveDays <= 0 {
		return nil, nil
	}

	consumeDate := consumeRewardDateAt(occurredAt)
	var reward *model.ConsumeRewardRecord
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		now := common.GetTimestamp()
		daily := model.ConsumeRewardDaily{
			UserId:       userId,
			ConsumeDate:  consumeDate,
			ConsumeQuota: quota,
			NetQuota:     quota,
			Qualified:    quota >= setting.DailyQuota,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}, {Name: "consume_date"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"consume_quota": gorm.Expr("? + ?", clause.Column{Table: "consume_reward_dailies", Name: "consume_quota"}, quota),
				"net_quota":     gorm.Expr("? + ?", clause.Column{Table: "consume_reward_dailies", Name: "net_quota"}, quota),
				"updated_at":    now,
			}),
		}).Create(&daily).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ? AND consume_date = ?", userId, consumeDate).First(&daily).Error; err != nil {
			return err
		}
		wasQualified := daily.NetQuota-quota >= setting.DailyQuota
		daily.Qualified = daily.NetQuota >= setting.DailyQuota
		if err := tx.Model(&model.ConsumeRewardDaily{}).Where("id = ?", daily.Id).Updates(map[string]interface{}{
			"qualified":  daily.Qualified,
			"updated_at": now,
		}).Error; err != nil {
			return err
		}
		if wasQualified || !daily.Qualified {
			return nil
		}

		grantedReward, err := grantConsumeRewardIfQualified(tx, userId, consumeDate, setting)
		if err != nil {
			return err
		}
		reward = grantedReward
		return nil
	})
	if err != nil {
		return nil, err
	}
	if reward == nil {
		return nil, nil
	}

	if err := model.InvalidateUserCache(userId); err != nil {
		common.SysLog(fmt.Sprintf("failed to invalidate user %d cache after consume reward: %v", userId, err))
	}
	model.RecordLog(userId, model.LogTypeSystem, fmt.Sprintf("连续消费奖励，获得额度 %s", logger.LogQuota(reward.RewardQuota)))
	return reward, nil
}

// RecordConsumeRewardRefund removes a refunded amount from the original
// consumption day. A reward already granted is never silently clawed back;
// administrators can review it through the reward ledger.
func RecordConsumeRewardRefund(userId int, quota int, consumeAt time.Time) error {
	setting := operation_setting.GetConsumeRewardSetting()
	if !setting.Enabled || quota <= 0 || setting.DailyQuota <= 0 {
		return nil
	}

	consumeDate := consumeRewardDateAt(consumeAt)
	return model.DB.Transaction(func(tx *gorm.DB) error {
		daily := model.ConsumeRewardDaily{}
		err := tx.Where("user_id = ? AND consume_date = ?", userId, consumeDate).First(&daily).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := tx.Model(&model.ConsumeRewardDaily{}).
			Where("id = ?", daily.Id).
			Updates(map[string]interface{}{
				"refund_quota": gorm.Expr("refund_quota + ?", quota),
				"net_quota":    gorm.Expr("net_quota - ?", quota),
				"updated_at":   common.GetTimestamp(),
			}).Error; err != nil {
			return err
		}
		if err := tx.Where("id = ?", daily.Id).First(&daily).Error; err != nil {
			return err
		}
		return tx.Model(&model.ConsumeRewardDaily{}).Where("id = ?", daily.Id).Updates(map[string]interface{}{
			"qualified":  daily.NetQuota >= setting.DailyQuota,
			"updated_at": common.GetTimestamp(),
		}).Error
	})
}

func grantConsumeRewardIfQualified(tx *gorm.DB, userId int, consumeDate string, setting *operation_setting.ConsumeRewardSetting) (*model.ConsumeRewardRecord, error) {
	lastAwardDate, err := lastAutomaticRewardDate(tx, userId)
	if err != nil {
		return nil, err
	}

	streak, err := consecutiveQualifiedDays(tx, userId, lastAwardDate, consumeDate)
	if err != nil || streak < setting.ConsecutiveDays {
		return nil, err
	}

	var rewardCount int64
	if err = tx.Model(&model.ConsumeRewardRecord{}).
		Where("user_id = ? AND source = ?", userId, consumeRewardSourceAutomatic).
		Count(&rewardCount).Error; err != nil {
		return nil, err
	}

	reward := &model.ConsumeRewardRecord{
		UserId:      userId,
		RewardKey:   "automatic:" + consumeDate,
		Cycle:       int(rewardCount) + 1,
		RewardDate:  consumeDate,
		RewardQuota: setting.RewardQuota,
		Source:      consumeRewardSourceAutomatic,
		Status:      consumeRewardStatusGranted,
		CreatedAt:   common.GetTimestamp(),
	}
	if err = tx.Create(reward).Error; err != nil {
		return nil, err
	}
	if err = tx.Model(&model.User{}).Where("id = ?", userId).
		UpdateColumn("quota", gorm.Expr("quota + ?", reward.RewardQuota)).Error; err != nil {
		return nil, err
	}
	return reward, nil
}

func consumeRewardDateAt(occurredAt time.Time) string {
	return occurredAt.In(consumeRewardLocation).Format(consumeRewardDateLayout)
}

func lastAutomaticRewardDate(db *gorm.DB, userId int) (string, error) {
	lastReward := model.ConsumeRewardRecord{}
	err := db.Where("user_id = ? AND source = ?", userId, consumeRewardSourceAutomatic).
		Order("reward_date DESC, id DESC").First(&lastReward).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return lastReward.RewardDate, nil
}

func consecutiveQualifiedDays(tx *gorm.DB, userId int, lastAwardDate string, endDate string) (int, error) {
	var dailies []model.ConsumeRewardDaily
	query := tx.Where("user_id = ? AND consume_date <= ?", userId, endDate)
	if lastAwardDate != "" {
		query = query.Where("consume_date > ?", lastAwardDate)
	}
	if err := query.Order("consume_date DESC").Find(&dailies).Error; err != nil {
		return 0, err
	}

	dailyByDate := make(map[string]model.ConsumeRewardDaily, len(dailies))
	for _, daily := range dailies {
		dailyByDate[daily.ConsumeDate] = daily
	}
	expected, err := time.ParseInLocation("2006-01-02", endDate, consumeRewardLocation)
	if err != nil {
		return 0, err
	}
	streak := 0
	for {
		date := expected.Format("2006-01-02")
		if lastAwardDate != "" && date <= lastAwardDate {
			break
		}
		daily, ok := dailyByDate[date]
		if !ok || !daily.Qualified {
			break
		}
		streak++
		expected = expected.AddDate(0, 0, -1)
	}
	return streak, nil
}
