package domain

import "github.com/kizuna-org/akari/gen/kiseki"

type PollingTask struct {
	TaskID   string
	TaskType string
	Data     *kiseki.PollingResponseItem_Data
}

type PollingTaskResult struct {
	TaskID string
	Data   *kiseki.PollingRequestItem_Data
}
