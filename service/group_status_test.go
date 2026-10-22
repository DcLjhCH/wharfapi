package service

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestGroupStatusWindowIsThirtyCompletedMinutes(t *testing.T) {
	t.Parallel()

	loc := time.FixedZone("CST", 8*60*60)
	now := time.Date(2026, 10, 22, 10, 23, 40, 0, loc)
	start, end := GroupStatusWindow(now.Unix())

	require.Equal(t, time.Date(2026, 10, 22, 9, 53, 0, 0, loc).Unix(), start)
	require.Equal(t, time.Date(2026, 10, 22, 10, 23, 0, 0, loc).Unix(), end)
	require.Equal(t, int64(30*60), end-start)

	// The same instant in UTC lands on the same unix minute boundary.
	utcStart, utcEnd := GroupStatusWindow(now.UTC().Unix())
	require.Equal(t, start, utcStart)
	require.Equal(t, end, utcEnd)

	exact := time.Date(2026, 10, 22, 10, 23, 0, 0, loc)
	exactStart, exactEnd := GroupStatusWindow(exact.Unix())
	require.Equal(t, exact.Unix(), exactEnd)
	require.Equal(t, exact.Add(-30*time.Minute).Unix(), exactStart)
}

func TestVisibleGroupNames(t *testing.T) {
	t.Parallel()

	ratios := map[string]float64{
		"default": 1,
		"vip":     2,
		"auto":    1,
		"":        1,
		"   ":     1,
		"hidden":  3,
	}
	usable := map[string]string{
		"default": "默认",
		"vip":     "贵宾",
		"auto":    "自动",
		"extra":   "不在倍率配置里",
	}

	require.Equal(t, []string{"default", "hidden", "vip"}, VisibleGroupNames(common.RoleAdminUser, ratios, usable))
	require.Equal(t, []string{"default", "hidden", "vip"}, VisibleGroupNames(common.RoleRootUser, ratios, usable))
	require.Equal(t, []string{"default", "vip"}, VisibleGroupNames(common.RoleCommonUser, ratios, usable))
	require.Equal(t, []string{"default", "vip"}, VisibleGroupNames(0, ratios, usable))
	require.Equal(t, []string{"default", "vip"}, VisibleGroupNames(common.RoleAdminUser-1, ratios, usable))
}

func TestGroupStatusDescription(t *testing.T) {
	t.Parallel()

	usable := map[string]string{"default": "默认分组", "blank": "   "}
	require.Equal(t, "默认分组", GroupStatusDescription("default", usable, "全局"))
	require.Equal(t, "全局", GroupStatusDescription("vip", usable, "全局"))
	require.Equal(t, "vip", GroupStatusDescription("vip", usable, "  "))
	require.Equal(t, "blank", GroupStatusDescription("blank", usable, "  "))
}

func TestAggregateGroupStatusEmptyIsNotHealthy(t *testing.T) {
	t.Parallel()

	start, end := GroupStatusWindow(time.Date(2026, 10, 22, 10, 23, 40, 0, time.UTC).Unix())
	data := AggregateGroupStatus(start, end, []GroupStatusCandidate{{
		Name:          "default",
		Description:   "  ",
		Ratio:         1.5,
		EnabledModels: []string{"gpt-4", "gpt-4", " "},
	}}, nil)

	require.Equal(t, start, data.WindowStart)
	require.Equal(t, end, data.WindowEnd)
	require.Len(t, data.Groups, 1)
	group := data.Groups[0]
	require.Equal(t, "default", group.Name)
	require.Equal(t, "default", group.Description)
	require.Equal(t, 1.5, group.Ratio)
	require.True(t, group.HasEnabledModels)
	require.Equal(t, 1, group.ModelCount)
	require.Equal(t, 1, group.ModelsWithoutRequests)
	require.Equal(t, 0, group.SuccessCount)
	require.Equal(t, 0, group.FailureCount)
	require.Nil(t, group.Availability)
	require.Nil(t, group.AvgResponseSeconds)
	require.Nil(t, group.AvgFirstTokenSeconds)
	require.Nil(t, group.CacheRate)
	require.Nil(t, group.LastSampleAt)
	require.Empty(t, group.Models)
	require.Len(t, group.History, 30)
	for _, status := range group.History {
		require.Equal(t, "empty", status)
	}
}

func TestAggregateGroupStatusRules(t *testing.T) {
	t.Parallel()

	start := int64(1_700_000_000)
	end := start + 30*60
	candidates := []GroupStatusCandidate{
		{Name: "auto", Description: "自动", Ratio: 0.1, EnabledModels: []string{"m"}},
		{Name: " ", Description: "空白", Ratio: 0.1, EnabledModels: []string{"m"}},
		{Name: "ghost", Description: "没有模型", Ratio: 0.1, EnabledModels: nil},
		{Name: "aaa", Description: "低倍率同值", Ratio: 1, EnabledModels: []string{"m"}},
		{Name: "default", Description: "默认分组", Ratio: 1, EnabledModels: []string{"a", "b", "d", "a"}},
		{Name: "vip", Description: "贵宾", Ratio: 2, EnabledModels: []string{"z"}},
	}
	logs := []GroupStatusLog{
		{Group: "default", Type: model.LogTypeConsume, ModelName: "b", UseTime: 10, PromptTokens: 100, Other: `{"cache_tokens":25,"frt":1500,"stream_status":{"status":"ok"}}`, CreatedAt: start + 10},
		{Group: "default", Type: model.LogTypeConsume, ModelName: "b", UseTime: 30, PromptTokens: 50, Other: `{"cache_tokens":0,"frt":0,"stream_status":{"status":"ok"}}`, CreatedAt: start + 20},
		{Group: "default", Type: model.LogTypeConsume, ModelName: "a", UseTime: 8, PromptTokens: 40, Other: `{"cache_tokens":10,"frt":500,"stream_status":{"status":"error"}}`, CreatedAt: start + 30},
		{Group: "default", Type: model.LogTypeError, ModelName: "a", UseTime: 100, PromptTokens: 10, Other: `{"cache_tokens":5,"frt":4000,"stream_status":{"status":"ok"}}`, CreatedAt: start + 70},
		{Group: "default", Type: model.LogTypeConsume, ModelName: "a", UseTime: 4, PromptTokens: 20, Other: `{bad`, CreatedAt: start + 80},
		{Group: "default", Type: model.LogTypeConsume, ModelName: "c", UseTime: 0, PromptTokens: 0, Other: `{"cache_tokens":7,"frt":-5}`, CreatedAt: end - 1},
		{Group: "default", Type: model.LogTypeConsume, ModelName: "b", UseTime: 999, PromptTokens: 999, Other: `{"cache_tokens":999,"frt":999}`, CreatedAt: start - 1},
		{Group: "default", Type: model.LogTypeConsume, ModelName: "b", UseTime: 999, PromptTokens: 999, Other: `{"cache_tokens":999,"frt":999}`, CreatedAt: end},
		{Group: "hidden", Type: model.LogTypeConsume, ModelName: "b", UseTime: 50, PromptTokens: 50, Other: `{"cache_tokens":50,"frt":5000}`, CreatedAt: start + 5},
		{Group: "auto", Type: model.LogTypeConsume, ModelName: "m", UseTime: 3, PromptTokens: 3, Other: `{"cache_tokens":1,"frt":1000}`, CreatedAt: start + 5},
		{Group: "default", Type: model.LogTypeTopup, ModelName: "b", UseTime: 9, PromptTokens: 9, Other: `{"cache_tokens":9,"frt":900}`, CreatedAt: start + 5},
	}

	data := AggregateGroupStatus(start, end, candidates, logs)
	require.Equal(t, []string{"aaa", "default", "vip"}, groupNames(data))

	aaa := data.Groups[0]
	require.Nil(t, aaa.Availability)
	require.Nil(t, aaa.AvgResponseSeconds)
	require.Nil(t, aaa.AvgFirstTokenSeconds)
	require.Nil(t, aaa.CacheRate)
	require.Nil(t, aaa.LastSampleAt)
	require.Equal(t, 1, aaa.ModelsWithoutRequests)
	require.Empty(t, aaa.Models)
	requireAllEmpty(t, aaa.History)

	group := data.Groups[1]
	require.Equal(t, "默认分组", group.Description)
	require.Equal(t, 4, group.SuccessCount)
	require.Equal(t, 2, group.FailureCount)
	requireFloat(t, group.Availability, 4.0/6.0)
	requireFloat(t, group.AvgResponseSeconds, 44.0/3.0)
	requireFloat(t, group.AvgFirstTokenSeconds, 1.5)
	requireFloat(t, group.CacheRate, 47.0/220.0)
	require.NotNil(t, group.LastSampleAt)
	require.Equal(t, end-1, *group.LastSampleAt)
	require.Equal(t, 3, group.ModelCount)
	require.Equal(t, 1, group.ModelsWithoutRequests)
	require.True(t, group.HasEnabledModels)

	require.Equal(t, "mixed", group.History[0])
	require.Equal(t, "mixed", group.History[1])
	require.Equal(t, "success", group.History[29])
	for i, status := range group.History {
		if i == 0 || i == 1 || i == 29 {
			continue
		}
		require.Equal(t, "empty", status)
	}

	require.Equal(t, []string{"a", "b", "c"}, modelNames(group))
	require.Equal(t, 3, group.Models[0].Requests)
	requireFloat(t, group.Models[0].Availability, 1.0/3.0)
	requireFloat(t, group.Models[0].AvgResponseSeconds, 4)
	require.Nil(t, group.Models[0].AvgFirstTokenSeconds)

	require.Equal(t, 2, group.Models[1].Requests)
	requireFloat(t, group.Models[1].Availability, 1)
	requireFloat(t, group.Models[1].AvgResponseSeconds, 20)
	requireFloat(t, group.Models[1].AvgFirstTokenSeconds, 1.5)

	require.Equal(t, 1, group.Models[2].Requests)
	requireFloat(t, group.Models[2].Availability, 1)
	require.Nil(t, group.Models[2].AvgResponseSeconds)
	require.Nil(t, group.Models[2].AvgFirstTokenSeconds)

	vip := data.Groups[2]
	require.Equal(t, 0, vip.SuccessCount)
	require.Equal(t, 0, vip.FailureCount)
	require.Nil(t, vip.Availability)
	require.Nil(t, vip.LastSampleAt)
	requireAllEmpty(t, vip.History)
}

func requireFloat(t *testing.T, got *float64, want float64) {
	t.Helper()
	require.NotNil(t, got)
	require.InDelta(t, want, *got, 1e-9)
}

func requireAllEmpty(t *testing.T, history []string) {
	t.Helper()
	require.Len(t, history, 30)
	for _, status := range history {
		require.Equal(t, "empty", status)
	}
}

func groupNames(data GroupStatusData) []string {
	names := make([]string, 0, len(data.Groups))
	for _, group := range data.Groups {
		names = append(names, group.Name)
	}
	return names
}

func modelNames(group GroupStatusGroup) []string {
	names := make([]string, 0, len(group.Models))
	for _, item := range group.Models {
		names = append(names, item.Name)
	}
	return names
}
