package model

// GroupStatusLogRow is the narrow log projection used by group status.
// Content, prompts, keys, IP, and channel identifiers are not selected.
type GroupStatusLogRow struct {
	Group        string `gorm:"column:group"`
	Type         int    `gorm:"column:type"`
	ModelName    string `gorm:"column:model_name"`
	UseTime      int    `gorm:"column:use_time"`
	PromptTokens int    `gorm:"column:prompt_tokens"`
	Other        string `gorm:"column:other"`
	CreatedAt    int64  `gorm:"column:created_at"`
}

// ListGroupStatusLogs reads consume and error logs for the given groups in
// [start, end). The end timestamp is excluded.
func ListGroupStatusLogs(groups []string, start, end int64) ([]GroupStatusLogRow, error) {
	rows := make([]GroupStatusLogRow, 0)
	if len(groups) == 0 || start >= end {
		return rows, nil
	}
	err := LOG_DB.Model(&Log{}).
		Select(logGroupCol+", type, model_name, use_time, prompt_tokens, other, created_at").
		Where("type IN ?", []int{LogTypeConsume, LogTypeError}).
		Where("created_at >= ? AND created_at < ?", start, end).
		Where(logGroupCol+" IN ?", groups).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}
