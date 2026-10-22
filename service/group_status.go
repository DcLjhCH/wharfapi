package service

import (
	"encoding/json"
	"math"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const groupStatusBucketCount = 30

const (
	groupStatusHistoryEmpty   = "empty"
	groupStatusHistorySuccess = "success"
	groupStatusHistoryFailure = "failure"
	groupStatusHistoryMixed   = "mixed"
)

// GroupStatusLog is the only log data used for group status.
// It intentionally has no request body, prompt, key, IP, or raw extension payload.
type GroupStatusLog struct {
	Group        string
	Type         int
	ModelName    string
	UseTime      int
	PromptTokens int
	Other        string
	CreatedAt    int64
}

// GroupStatusCandidate is one group the current user is allowed to see.
type GroupStatusCandidate struct {
	Name          string
	Description   string
	Ratio         float64
	EnabledModels []string
}

type GroupStatusModel struct {
	Name                 string   `json:"name"`
	Requests             int      `json:"requests"`
	Availability         *float64 `json:"availability"`
	AvgResponseSeconds   *float64 `json:"avg_response_seconds"`
	AvgFirstTokenSeconds *float64 `json:"avg_first_token_seconds"`
}

type GroupStatusGroup struct {
	Name                  string             `json:"name"`
	Description           string             `json:"description"`
	Ratio                 float64            `json:"ratio"`
	HasEnabledModels      bool               `json:"has_enabled_models"`
	ModelCount            int                `json:"model_count"`
	ModelsWithoutRequests int                `json:"models_without_requests"`
	SuccessCount          int                `json:"success_count"`
	FailureCount          int                `json:"failure_count"`
	Availability          *float64           `json:"availability"`
	AvgResponseSeconds    *float64           `json:"avg_response_seconds"`
	AvgFirstTokenSeconds  *float64           `json:"avg_first_token_seconds"`
	CacheRate             *float64           `json:"cache_rate"`
	LastSampleAt          *int64             `json:"last_sample_at"`
	History               []string           `json:"history"`
	Models                []GroupStatusModel `json:"models"`
}

type GroupStatusData struct {
	WindowStart int64              `json:"window_start"`
	WindowEnd   int64              `json:"window_end"`
	Groups      []GroupStatusGroup `json:"groups"`
}

// GroupStatusWindow is the 30 completed minute buckets ending at the current minute.
// The end is exclusive. At 10:23:40 the range is [09:53:00, 10:23:00).
func GroupStatusWindow(nowUnix int64) (start int64, end int64) {
	end = nowUnix - nowUnix%60
	start = end - groupStatusBucketCount*60
	return start, end
}

// VisibleGroupNames returns ratio groups the role may view.
// Admins see every configured ratio group. Other users see the intersection of
// their usable groups and the ratio groups. Blank names and auto are removed.
func VisibleGroupNames(role int, ratioGroups map[string]float64, usable map[string]string) []string {
	names := make([]string, 0)
	if role >= common.RoleAdminUser {
		for name := range ratioGroups {
			if !skipGroupStatusName(name) {
				names = append(names, name)
			}
		}
	} else {
		for name := range usable {
			if _, ok := ratioGroups[name]; !ok || skipGroupStatusName(name) {
				continue
			}
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// GroupStatusDescription prefers the caller's usable-group description.
// fallback is the global usable-group description. The group name is last.
func GroupStatusDescription(name string, usable map[string]string, fallback string) string {
	if usable != nil {
		if desc := strings.TrimSpace(usable[name]); desc != "" {
			return desc
		}
	}
	if desc := strings.TrimSpace(fallback); desc != "" {
		return desc
	}
	return name
}

func skipGroupStatusName(name string) bool {
	trimmed := strings.TrimSpace(name)
	return trimmed == "" || trimmed == "auto"
}

// AggregateGroupStatus summarizes already loaded logs. Logs outside
// [start, end), logs for groups that are not candidates, and log types other
// than consume/error are ignored. A group with no enabled model is omitted.
// No traffic stays null and empty; it is not reported as healthy.
func AggregateGroupStatus(start, end int64, candidates []GroupStatusCandidate, logs []GroupStatusLog) GroupStatusData {
	prepared := make([]preparedGroup, 0, len(candidates))
	index := make(map[string]int, len(candidates))
	for _, candidate := range candidates {
		if skipGroupStatusName(candidate.Name) {
			continue
		}
		enabled := uniqueModelNames(candidate.EnabledModels)
		if len(enabled) == 0 {
			continue
		}
		if _, exists := index[candidate.Name]; exists {
			continue
		}
		description := strings.TrimSpace(candidate.Description)
		if description == "" {
			description = candidate.Name
		}
		index[candidate.Name] = len(prepared)
		prepared = append(prepared, preparedGroup{
			name:        candidate.Name,
			description: description,
			ratio:       candidate.Ratio,
			enabled:     enabled,
			models:      map[string]*statusAccumulator{},
		})
	}

	for _, log := range logs {
		if log.CreatedAt < start || log.CreatedAt >= end {
			continue
		}
		pos, ok := index[log.Group]
		if !ok {
			continue
		}
		if log.Type != model.LogTypeConsume && log.Type != model.LogTypeError {
			continue
		}
		bucket := int((log.CreatedAt - start) / 60)
		if bucket < 0 || bucket >= groupStatusBucketCount {
			continue
		}
		prepared[pos].add(log, bucket)
	}

	groups := make([]GroupStatusGroup, 0, len(prepared))
	for _, group := range prepared {
		groups = append(groups, group.snapshot())
	}
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].Ratio != groups[j].Ratio {
			return groups[i].Ratio < groups[j].Ratio
		}
		return groups[i].Name < groups[j].Name
	})

	return GroupStatusData{
		WindowStart: start,
		WindowEnd:   end,
		Groups:      groups,
	}
}

type preparedGroup struct {
	name        string
	description string
	ratio       float64
	enabled     []string
	statusAccumulator
	models map[string]*statusAccumulator
}

func (group *preparedGroup) add(log GroupStatusLog, bucket int) {
	extension := parseLogExtension(log.Other)
	failure := log.Type == model.LogTypeError || extension.streamError
	group.statusAccumulator.add(log, extension, failure, bucket)

	modelName := log.ModelName
	acc := group.models[modelName]
	if acc == nil {
		acc = &statusAccumulator{}
		group.models[modelName] = acc
	}
	acc.add(log, extension, failure, bucket)
}

func (group preparedGroup) snapshot() GroupStatusGroup {
	models := make([]GroupStatusModel, 0, len(group.models))
	seen := make(map[string]struct{}, len(group.models))
	for name, acc := range group.models {
		seen[name] = struct{}{}
		availability, response, firstToken := acc.rates()
		models = append(models, GroupStatusModel{
			Name:                 name,
			Requests:             acc.success + acc.failure,
			Availability:         availability,
			AvgResponseSeconds:   response,
			AvgFirstTokenSeconds: firstToken,
		})
	}
	sort.Slice(models, func(i, j int) bool {
		if models[i].Requests != models[j].Requests {
			return models[i].Requests > models[j].Requests
		}
		return models[i].Name < models[j].Name
	})

	missing := 0
	for _, name := range group.enabled {
		if _, ok := seen[name]; !ok {
			missing++
		}
	}
	availability, response, firstToken := group.rates()
	history := make([]string, groupStatusBucketCount)
	for i, bucket := range group.buckets {
		history[i] = bucket.status()
	}

	return GroupStatusGroup{
		Name:                  group.name,
		Description:           group.description,
		Ratio:                 group.ratio,
		HasEnabledModels:      true,
		ModelCount:            len(group.enabled),
		ModelsWithoutRequests: missing,
		SuccessCount:          group.success,
		FailureCount:          group.failure,
		Availability:          availability,
		AvgResponseSeconds:    response,
		AvgFirstTokenSeconds:  firstToken,
		CacheRate:             cacheRate(group.cacheTokens, group.promptTokens),
		LastSampleAt:          group.lastSampleAt,
		History:               history,
		Models:                models,
	}
}

type statusAccumulator struct {
	success      int
	failure      int
	responseSum  float64
	responseN    int
	firstToken   float64
	firstTokenN  int
	cacheTokens  int64
	promptTokens int64
	lastSampleAt *int64
	buckets      [groupStatusBucketCount]statusBucket
}

func (acc *statusAccumulator) add(log GroupStatusLog, extension logExtension, failure bool, bucket int) {
	if failure {
		acc.failure++
		acc.buckets[bucket].failure++
	} else {
		acc.success++
		acc.buckets[bucket].success++
		if log.UseTime > 0 {
			acc.responseSum += float64(log.UseTime)
			acc.responseN++
		}
		if extension.frtValid {
			acc.firstToken += extension.frtSeconds
			acc.firstTokenN++
		}
	}
	acc.cacheTokens += extension.cacheTokens
	if log.PromptTokens > 0 {
		acc.promptTokens += int64(log.PromptTokens)
	}
	if acc.lastSampleAt == nil || log.CreatedAt > *acc.lastSampleAt {
		created := log.CreatedAt
		acc.lastSampleAt = &created
	}
}

func (acc statusAccumulator) rates() (availability *float64, response *float64, firstToken *float64) {
	total := acc.success + acc.failure
	if total > 0 {
		value := float64(acc.success) / float64(total)
		availability = &value
	}
	if acc.responseN > 0 {
		value := acc.responseSum / float64(acc.responseN)
		response = &value
	}
	if acc.firstTokenN > 0 {
		value := acc.firstToken / float64(acc.firstTokenN)
		firstToken = &value
	}
	return availability, response, firstToken
}

type statusBucket struct {
	success int
	failure int
}

func (bucket statusBucket) status() string {
	switch {
	case bucket.success == 0 && bucket.failure == 0:
		return groupStatusHistoryEmpty
	case bucket.failure == 0:
		return groupStatusHistorySuccess
	case bucket.success == 0:
		return groupStatusHistoryFailure
	default:
		return groupStatusHistoryMixed
	}
}

func cacheRate(cacheTokens, promptTokens int64) *float64 {
	if promptTokens <= 0 {
		return nil
	}
	value := float64(cacheTokens) / float64(promptTokens)
	return &value
}

func uniqueModelNames(names []string) []string {
	if len(names) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(names))
	unique := make([]string, 0, len(names))
	for _, name := range names {
		if strings.TrimSpace(name) == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		unique = append(unique, name)
	}
	return unique
}

type logExtension struct {
	cacheTokens int64
	frtSeconds  float64
	frtValid    bool
	streamError bool
}

func parseLogExtension(raw string) logExtension {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return logExtension{}
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil || payload == nil {
		return logExtension{}
	}
	extension := logExtension{streamError: streamStatusIsError(payload["stream_status"])}
	if value, ok := positiveFloat(payload["cache_tokens"]); ok {
		rounded := math.Round(value)
		if rounded > 0 && rounded < math.MaxInt64 {
			extension.cacheTokens = int64(rounded)
		}
	}
	if value, ok := positiveFloat(payload["frt"]); ok {
		extension.frtSeconds = value / 1000
		extension.frtValid = true
	}
	return extension
}

func streamStatusIsError(value any) bool {
	status, ok := value.(map[string]any)
	if !ok {
		return false
	}
	text, ok := status["status"].(string)
	return ok && text == "error"
}

func positiveFloat(value any) (float64, bool) {
	number, ok := value.(float64)
	if !ok || math.IsNaN(number) || math.IsInf(number, 0) || number <= 0 {
		return 0, false
	}
	return number, true
}
