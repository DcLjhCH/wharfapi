package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func configureInviteRedemptionReward(t *testing.T, multiplier float64, startTime int64) {
	t.Helper()
	setting := operation_setting.GetInviteRedemptionRewardSetting()
	previous := *setting
	*setting = operation_setting.InviteRedemptionRewardSetting{
		Enabled:    true,
		Multiplier: multiplier,
		StartTime:  startTime,
	}
	t.Cleanup(func() { *setting = previous })
}

func TestRedeemBalanceCodeAddsQuota(t *testing.T) {
	truncateTables(t)

	insertUserForPaymentGuardTest(t, 501, 100)
	code := &Redemption{
		UserId:      1,
		Key:         "balance-redemption-key",
		Status:      common.RedemptionCodeStatusEnabled,
		Name:        "Balance Code",
		Type:        RedemptionTypeBalance,
		Quota:       250,
		CreatedTime: common.GetTimestamp(),
	}
	require.NoError(t, code.Insert())

	result, err := Redeem(code.Key, 501)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, RedemptionTypeBalance, result.Type)
	assert.Equal(t, 250, result.Quota)
	assert.Zero(t, result.PlanId)
	assert.Zero(t, result.SubscriptionId)
	assert.Equal(t, 350, getUserQuotaForPaymentGuardTest(t, 501))
	assert.Zero(t, countUserSubscriptionsForPaymentGuardTest(t, 501))
}

func TestRedeemSubscriptionCodeCreatesActiveSubscriptionWithoutAddingQuota(t *testing.T) {
	truncateTables(t)

	insertUserForPaymentGuardTest(t, 502, 100)
	plan := insertSubscriptionPlanForPaymentGuardTest(t, 601)
	code := &Redemption{
		UserId:      1,
		Key:         "subscription-redemption-key",
		Status:      common.RedemptionCodeStatusEnabled,
		Name:        "Subscription Code",
		Type:        RedemptionTypeSubscription,
		PlanId:      plan.Id,
		Quota:       999,
		CreatedTime: common.GetTimestamp(),
	}
	require.NoError(t, code.Insert())

	result, err := Redeem(code.Key, 502)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, RedemptionTypeSubscription, result.Type)
	assert.Zero(t, result.Quota)
	assert.Equal(t, plan.Id, result.PlanId)
	assert.NotZero(t, result.SubscriptionId)
	assert.Equal(t, 100, getUserQuotaForPaymentGuardTest(t, 502))

	var subscription UserSubscription
	require.NoError(t, DB.Where("id = ?", result.SubscriptionId).First(&subscription).Error)
	assert.Equal(t, 502, subscription.UserId)
	assert.Equal(t, plan.Id, subscription.PlanId)
	assert.Equal(t, "active", subscription.Status)
	assert.Equal(t, "redemption", subscription.Source)

	var redeemed Redemption
	require.NoError(t, DB.Where("id = ?", code.Id).First(&redeemed).Error)
	assert.Equal(t, common.RedemptionCodeStatusUsed, redeemed.Status)
	assert.Equal(t, 502, redeemed.UsedUserId)
	assert.NotZero(t, redeemed.RedeemedTime)
}

func TestRedeemBalanceCodeAwardsEligibleInviterToAffiliateQuota(t *testing.T) {
	truncateTables(t)
	startTime := time.Now().Unix()
	configureInviteRedemptionReward(t, 0.25, startTime)

	inviter := &User{Id: 510, Username: "reward-inviter", Status: common.UserStatusEnabled}
	invitee := &User{
		Id:        511,
		Username:  "reward-invitee",
		Status:    common.UserStatusEnabled,
		InviterId: inviter.Id,
		CreatedAt: startTime,
	}
	require.NoError(t, DB.Create(inviter).Error)
	require.NoError(t, DB.Create(invitee).Error)
	code := &Redemption{
		UserId:      1,
		Key:         "reward-balance-redemption-key",
		Status:      common.RedemptionCodeStatusEnabled,
		Name:        "Reward Balance Code",
		Type:        RedemptionTypeBalance,
		Quota:       400,
		CreatedTime: startTime,
	}
	require.NoError(t, code.Insert())

	_, err := Redeem(code.Key, invitee.Id)
	require.NoError(t, err)

	var reloadedInviter User
	require.NoError(t, DB.First(&reloadedInviter, inviter.Id).Error)
	assert.Equal(t, 100, reloadedInviter.AffQuota)
	assert.Equal(t, 100, reloadedInviter.AffHistoryQuota)
	assert.Equal(t, 400, getUserQuotaForPaymentGuardTest(t, invitee.Id))
}

func TestRedeemBalanceCodeDoesNotRewardInvitesFromBeforeActivityStart(t *testing.T) {
	truncateTables(t)
	startTime := time.Now().Unix()
	configureInviteRedemptionReward(t, 1, startTime)

	inviter := &User{Id: 520, Username: "legacy-inviter", Status: common.UserStatusEnabled}
	invitee := &User{
		Id:        521,
		Username:  "legacy-invitee",
		Status:    common.UserStatusEnabled,
		InviterId: inviter.Id,
		CreatedAt: startTime - 1,
	}
	require.NoError(t, DB.Create(inviter).Error)
	require.NoError(t, DB.Create(invitee).Error)
	code := &Redemption{
		UserId:      1,
		Key:         "legacy-balance-redemption-key",
		Status:      common.RedemptionCodeStatusEnabled,
		Name:        "Legacy Balance Code",
		Type:        RedemptionTypeBalance,
		Quota:       400,
		CreatedTime: startTime,
	}
	require.NoError(t, code.Insert())

	_, err := Redeem(code.Key, invitee.Id)
	require.NoError(t, err)

	var reloadedInviter User
	require.NoError(t, DB.First(&reloadedInviter, inviter.Id).Error)
	assert.Zero(t, reloadedInviter.AffQuota)
	assert.Zero(t, reloadedInviter.AffHistoryQuota)
}

func TestRegistrationDoesNotGrantLegacyInvitationQuota(t *testing.T) {
	truncateTables(t)
	previousInviterQuota := common.QuotaForInviter
	previousInviteeQuota := common.QuotaForInvitee
	common.QuotaForInviter = 500
	common.QuotaForInvitee = 300
	t.Cleanup(func() {
		common.QuotaForInviter = previousInviterQuota
		common.QuotaForInvitee = previousInviteeQuota
	})

	inviter := &User{Id: 530, Username: "registration-inviter", Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(inviter).Error)
	invitee := &User{
		Username:  "registration-invitee",
		Password:  "password123",
		Status:    common.UserStatusEnabled,
		InviterId: inviter.Id,
	}
	require.NoError(t, invitee.Insert(inviter.Id))

	var reloadedInviter User
	require.NoError(t, DB.First(&reloadedInviter, inviter.Id).Error)
	assert.Equal(t, 1, reloadedInviter.AffCount)
	assert.Zero(t, reloadedInviter.AffQuota)
	assert.Zero(t, reloadedInviter.AffHistoryQuota)
}
