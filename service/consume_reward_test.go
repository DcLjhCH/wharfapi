package service

import (
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestConsumeRewardGrantsEverySevenConsecutiveQualifiedDays(t *testing.T) {
	truncate(t)
	configureConsumeReward(t, 100, 25)
	seedUser(t, 701, 0)
	start := time.Date(2026, time.July, 1, 12, 0, 0, 0, time.UTC)

	for day := 0; day < 14; day++ {
		reward, err := RecordConsumeRewardConsumption(701, 100, start.AddDate(0, 0, day))
		require.NoError(t, err)
		if day == 6 || day == 13 {
			require.NotNil(t, reward)
			require.Equal(t, 25, reward.RewardQuota)
		} else {
			require.Nil(t, reward)
		}
	}

	var records []model.ConsumeRewardRecord
	require.NoError(t, model.DB.Where("user_id = ?", 701).Order("cycle ASC").Find(&records).Error)
	require.Len(t, records, 2)
	require.Equal(t, 1, records[0].Cycle)
	require.Equal(t, 2, records[1].Cycle)
	require.Equal(t, 50, getUserQuota(t, 701))
}

func TestConsumeRewardBreaksTheStreakAfterAnUnqualifiedDay(t *testing.T) {
	truncate(t)
	configureConsumeReward(t, 100, 25)
	seedUser(t, 702, 0)
	start := time.Date(2026, time.July, 1, 12, 0, 0, 0, time.UTC)

	for day := 0; day < 6; day++ {
		_, err := RecordConsumeRewardConsumption(702, 100, start.AddDate(0, 0, day))
		require.NoError(t, err)
	}
	_, err := RecordConsumeRewardConsumption(702, 100, start.AddDate(0, 0, 7))
	require.NoError(t, err)

	var count int64
	require.NoError(t, model.DB.Model(&model.ConsumeRewardRecord{}).Where("user_id = ?", 702).Count(&count).Error)
	require.Zero(t, count)
}

func TestConsumeRewardRefundCanMakePendingDayUnqualified(t *testing.T) {
	truncate(t)
	configureConsumeReward(t, 100, 25)
	seedUser(t, 703, 0)
	start := time.Date(2026, time.July, 1, 12, 0, 0, 0, time.UTC)

	for day := 0; day < 6; day++ {
		_, err := RecordConsumeRewardConsumption(703, 100, start.AddDate(0, 0, day))
		require.NoError(t, err)
	}
	require.NoError(t, RecordConsumeRewardRefund(703, 100, start))
	reward, err := RecordConsumeRewardConsumption(703, 100, start.AddDate(0, 0, 6))
	require.NoError(t, err)
	require.Nil(t, reward)
}

func TestManualConsumeRewardGrantAndRevoke(t *testing.T) {
	truncate(t)
	seedUser(t, 704, 100)
	reward, err := GrantConsumeRewardManually(704, 25, 1, "support correction")
	require.NoError(t, err)
	require.Equal(t, 125, getUserQuota(t, 704))
	require.NoError(t, RevokeConsumeReward(reward.Id, 1, "duplicate grant"))
	require.Equal(t, 100, getUserQuota(t, 704))

	var persisted model.ConsumeRewardRecord
	require.NoError(t, model.DB.First(&persisted, reward.Id).Error)
	require.Equal(t, consumeRewardStatusRevoked, persisted.Status)
}

func TestConsumeRewardConcurrentThresholdCrossingGrantsOnce(t *testing.T) {
	truncate(t)
	configureConsumeReward(t, 100, 25)
	seedUser(t, 705, 0)
	start := time.Date(2026, time.July, 1, 12, 0, 0, 0, time.UTC)
	for day := 0; day < 6; day++ {
		_, err := RecordConsumeRewardConsumption(705, 100, start.AddDate(0, 0, day))
		require.NoError(t, err)
	}

	var waitGroup sync.WaitGroup
	errors := make(chan error, 2)
	for range 2 {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			_, err := RecordConsumeRewardConsumption(705, 50, start.AddDate(0, 0, 6))
			errors <- err
		}()
	}
	waitGroup.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}

	var records []model.ConsumeRewardRecord
	require.NoError(t, model.DB.Where("user_id = ?", 705).Find(&records).Error)
	require.Len(t, records, 1)
	require.Equal(t, 25, getUserQuota(t, 705))

	var daily model.ConsumeRewardDaily
	require.NoError(t, model.DB.Where("user_id = ? AND consume_date = ?", 705, "2026-07-07").First(&daily).Error)
	require.Equal(t, 100, daily.NetQuota)
	require.True(t, daily.Qualified)
}

func TestConsumeRewardUsesShanghaiCalendarDay(t *testing.T) {
	truncate(t)
	configureConsumeReward(t, 100, 25)
	seedUser(t, 706, 0)
	beforeMidnightShanghai := time.Date(2026, time.July, 1, 15, 59, 0, 0, time.UTC)
	afterMidnightShanghai := time.Date(2026, time.July, 1, 16, 0, 0, 0, time.UTC)

	_, err := RecordConsumeRewardConsumption(706, 100, beforeMidnightShanghai)
	require.NoError(t, err)
	_, err = RecordConsumeRewardConsumption(706, 100, afterMidnightShanghai)
	require.NoError(t, err)

	var dailies []model.ConsumeRewardDaily
	require.NoError(t, model.DB.Where("user_id = ?", 706).Order("consume_date ASC").Find(&dailies).Error)
	require.Len(t, dailies, 2)
	require.Equal(t, "2026-07-01", dailies[0].ConsumeDate)
	require.Equal(t, "2026-07-02", dailies[1].ConsumeDate)
}

func TestConsumeRewardRefundDoesNotAutomaticallyRevokeGrantedReward(t *testing.T) {
	truncate(t)
	configureConsumeReward(t, 100, 25)
	seedUser(t, 707, 0)
	start := time.Date(2026, time.July, 1, 12, 0, 0, 0, time.UTC)
	for day := 0; day < 7; day++ {
		_, err := RecordConsumeRewardConsumption(707, 100, start.AddDate(0, 0, day))
		require.NoError(t, err)
	}

	require.NoError(t, RecordConsumeRewardRefund(707, 100, start.AddDate(0, 0, 6)))
	require.Equal(t, 25, getUserQuota(t, 707))

	var reward model.ConsumeRewardRecord
	require.NoError(t, model.DB.Where("user_id = ?", 707).First(&reward).Error)
	require.Equal(t, consumeRewardStatusGranted, reward.Status)

	var daily model.ConsumeRewardDaily
	require.NoError(t, model.DB.Where("user_id = ? AND consume_date = ?", 707, "2026-07-07").First(&daily).Error)
	require.False(t, daily.Qualified)
}

func TestConsumeRewardRevocationKeepsTheAutomaticCycleBoundary(t *testing.T) {
	truncate(t)
	configureConsumeReward(t, 100, 25)
	seedUser(t, 708, 0)
	start := time.Date(2026, time.July, 1, 12, 0, 0, 0, time.UTC)

	var granted *model.ConsumeRewardRecord
	for day := 0; day < 7; day++ {
		reward, err := RecordConsumeRewardConsumption(708, 100, start.AddDate(0, 0, day))
		require.NoError(t, err)
		if reward != nil {
			granted = reward
		}
	}
	require.NotNil(t, granted)
	require.NoError(t, RevokeConsumeReward(granted.Id, 1, "refund correction"))

	reward, err := RecordConsumeRewardConsumption(708, 100, start.AddDate(0, 0, 7))
	require.NoError(t, err)
	require.Nil(t, reward, "a revoked automatic reward still starts the next cycle")

	var records []model.ConsumeRewardRecord
	require.NoError(t, model.DB.Where("user_id = ?", 708).Find(&records).Error)
	require.Len(t, records, 1)
	require.Equal(t, consumeRewardStatusRevoked, records[0].Status)
}

func TestTaskConsumeRewardDateKeepsRefundOnTheSettlementDay(t *testing.T) {
	truncate(t)
	configureConsumeReward(t, 100, 25)
	seedUser(t, 711, 0)
	task := &model.Task{
		CreatedAt:         time.Date(2026, time.July, 1, 15, 59, 0, 0, time.UTC).Unix(),
		ConsumeRewardDate: "2026-07-02",
	}

	_, err := RecordConsumeRewardConsumption(711, 100, taskConsumeRewardOccurredAt(task))
	require.NoError(t, err)
	require.NoError(t, RecordConsumeRewardRefund(711, 100, taskConsumeRewardOccurredAt(task)))

	var daily model.ConsumeRewardDaily
	require.NoError(t, model.DB.Where("user_id = ? AND consume_date = ?", 711, "2026-07-02").First(&daily).Error)
	require.Zero(t, daily.NetQuota)
	require.False(t, daily.Qualified)

	var previousDayCount int64
	require.NoError(t, model.DB.Model(&model.ConsumeRewardDaily{}).
		Where("user_id = ? AND consume_date = ?", 711, "2026-07-01").Count(&previousDayCount).Error)
	require.Zero(t, previousDayCount)
}

func TestGetConsumeRewardProgressReturnsTheCurrentUsersCycle(t *testing.T) {
	truncate(t)
	configureConsumeReward(t, 100, 25)
	seedUser(t, 709, 0)
	require.NoError(t, model.DB.Create(&model.User{
		Id:       710,
		Username: "other_user",
		AffCode:  "progress-other",
		Status:   common.UserStatusEnabled,
	}).Error)
	start := time.Date(2026, time.July, 1, 12, 0, 0, 0, time.UTC)

	for day := 0; day < 9; day++ {
		_, err := RecordConsumeRewardConsumption(709, 100, start.AddDate(0, 0, day))
		require.NoError(t, err)
	}
	_, err := GrantConsumeRewardManually(710, 50, 1, "other user")
	require.NoError(t, err)
	progress, err := GetConsumeRewardProgress(709, start.AddDate(0, 0, 8))
	require.NoError(t, err)
	require.True(t, progress.Enabled)
	require.Equal(t, "2026-07-09", progress.Today)
	require.Equal(t, 100, progress.TodayNetQuota)
	require.True(t, progress.TodayQualified)
	require.Equal(t, 2, progress.CurrentStreak)
	require.Equal(t, 5, progress.RemainingDays)
	require.Equal(t, "2026-07", progress.CalendarMonth)
	require.Equal(t, 2, progress.CalendarStartWeekday)
	require.Len(t, progress.CalendarDays, 31)
	require.Equal(t, "2026-07-01", progress.CalendarDays[0].ConsumeDate)
	require.Equal(t, "2026-07-31", progress.CalendarDays[30].ConsumeDate)
	require.Len(t, progress.Rewards, 1)
	require.Equal(t, "2026-07-07", progress.Rewards[0].RewardDate)
	require.Equal(t, consumeRewardSourceAutomatic, progress.Rewards[0].Source)
}
