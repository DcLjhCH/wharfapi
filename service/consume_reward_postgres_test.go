package service

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestConsumeRewardPostgresConcurrentThresholdCrossing is an opt-in
// integration test. Set NEWAPI_TEST_POSTGRES_DSN to run it against a disposable
// PostgreSQL instance. It exercises concurrent upserts and the unique reward
// ledger constraint with separate database connections.
func TestConsumeRewardPostgresConcurrentThresholdCrossing(t *testing.T) {
	dsn := os.Getenv("NEWAPI_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("NEWAPI_TEST_POSTGRES_DSN is not configured")
	}
	redisURL := os.Getenv("NEWAPI_TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("NEWAPI_TEST_REDIS_URL is not configured")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(8)
	t.Cleanup(func() { _ = sqlDB.Close() })

	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousMainType, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()
	model.DB, model.LOG_DB = db, db
	common.SetDatabaseTypes(common.DatabaseTypePostgreSQL, common.DatabaseTypePostgreSQL)
	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainType, previousLogType)
	})

	previousRedisClient, previousRedisEnabled := common.RDB, common.RedisEnabled
	previousRedisURL, hadRedisURL := os.LookupEnv("REDIS_CONN_STRING")
	require.NoError(t, os.Setenv("REDIS_CONN_STRING", redisURL))
	common.RedisEnabled = true
	require.NoError(t, common.InitRedisClient())
	t.Cleanup(func() {
		if common.RDB != nil {
			_ = common.RDB.Close()
		}
		common.RDB, common.RedisEnabled = previousRedisClient, previousRedisEnabled
		if hadRedisURL {
			_ = os.Setenv("REDIS_CONN_STRING", previousRedisURL)
		} else {
			_ = os.Unsetenv("REDIS_CONN_STRING")
		}
	})

	require.NoError(t, db.Migrator().DropTable(
		&model.ConsumeRewardRecord{},
		&model.ConsumeRewardDaily{},
		&model.Log{},
		&model.User{},
	))
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Log{},
		&model.ConsumeRewardDaily{},
		&model.ConsumeRewardRecord{},
	))

	configureConsumeReward(t, 100, 25)
	seedUser(t, 708, 0)
	start := time.Date(2026, time.July, 1, 12, 0, 0, 0, time.UTC)
	for day := 0; day < 6; day++ {
		_, err := RecordConsumeRewardConsumption(708, 100, start.AddDate(0, 0, day))
		require.NoError(t, err)
	}
	require.NoError(t, common.RedisSet("user:708", "stale cache", 0))

	startCalls := make(chan struct{})
	errors := make(chan error, 2)
	var waitGroup sync.WaitGroup
	for range 2 {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			<-startCalls
			_, err := RecordConsumeRewardConsumption(708, 50, start.AddDate(0, 0, 6))
			errors <- err
		}()
	}
	close(startCalls)
	waitGroup.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}

	var rewards []model.ConsumeRewardRecord
	require.NoError(t, db.Where("user_id = ?", 708).Find(&rewards).Error)
	require.Len(t, rewards, 1)
	require.Equal(t, "automatic:2026-07-07", rewards[0].RewardKey)
	require.Equal(t, 25, getUserQuota(t, 708))
	_, err = common.RedisGet("user:708")
	require.Error(t, err, "the reward grant must invalidate the user cache")
}

