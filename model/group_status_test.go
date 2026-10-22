package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListGroupStatusLogsFiltersWindowAndGroups(t *testing.T) {
	truncateTables(t)

	start := int64(1_700_000_000)
	end := start + 30*60
	logs := []Log{
		{Group: "default", Type: LogTypeConsume, ModelName: "a", UseTime: 3, PromptTokens: 11, Other: `{"cache_tokens":1}`, CreatedAt: start, Content: "SECRET_PROMPT", Ip: "10.0.0.8", TokenName: "secret-key"},
		{Group: "default", Type: LogTypeError, ModelName: "b", UseTime: 9, PromptTokens: 4, Other: `{"stream_status":{"status":"error"}}`, CreatedAt: end - 1, Content: "SECRET_ERROR"},
		{Group: "default", Type: LogTypeConsume, ModelName: "late", UseTime: 100, PromptTokens: 100, CreatedAt: end, Content: "OUTSIDE"},
		{Group: "default", Type: LogTypeConsume, ModelName: "early", UseTime: 100, PromptTokens: 100, CreatedAt: start - 1, Content: "OUTSIDE"},
		{Group: "hidden", Type: LogTypeConsume, ModelName: "a", UseTime: 8, PromptTokens: 8, CreatedAt: start + 10, Content: "HIDDEN"},
		{Group: "default", Type: LogTypeTopup, ModelName: "a", UseTime: 1, PromptTokens: 1, CreatedAt: start + 10, Content: "TOPUP"},
		{Group: "auto", Type: LogTypeConsume, ModelName: "a", UseTime: 1, PromptTokens: 1, CreatedAt: start + 10, Content: "AUTO"},
	}
	for i := range logs {
		require.NoError(t, LOG_DB.Create(&logs[i]).Error)
	}

	rows, err := ListGroupStatusLogs([]string{"default"}, start, end)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	got := map[string]GroupStatusLogRow{}
	for _, row := range rows {
		got[row.ModelName] = row
	}
	require.Equal(t, LogTypeConsume, got["a"].Type)
	require.Equal(t, 3, got["a"].UseTime)
	require.Equal(t, 11, got["a"].PromptTokens)
	require.Equal(t, start, got["a"].CreatedAt)
	require.JSONEq(t, `{"cache_tokens":1}`, got["a"].Other)
	require.Equal(t, LogTypeError, got["b"].Type)
	require.Equal(t, end-1, got["b"].CreatedAt)
	require.NotContains(t, got["a"].Other, "SECRET")
	require.NotContains(t, got["b"].Other, "SECRET")

	empty, err := ListGroupStatusLogs(nil, start, end)
	require.NoError(t, err)
	require.Empty(t, empty)
}
