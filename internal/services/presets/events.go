package presets

import "time"

const (
	EventTopicStressSender   = "presets.topic_stress.sender"
	EventTopicStressReceiver = "presets.topic_stress.receiver"
)

type TopicStressConfig struct {
	SourceID    uint32 `json:"sourceId"`
	TargetID    uint32 `json:"targetId"`
	Topic       string `json:"topic"`
	RunID       string `json:"runId"`
	Total       int    `json:"total"`
	PayloadSize int    `json:"payloadSize"`
	MaxPerSec   int    `json:"maxPerSec"`
}

type TopicStressSenderStatus struct {
	Active      bool      `json:"active"`
	Topic       string    `json:"topic"`
	RunID       string    `json:"runId"`
	Total       int       `json:"total"`
	PayloadSize int       `json:"payloadSize"`
	MaxPerSec   int       `json:"maxPerSec"`
	Sent        int       `json:"sent"`
	Errors      int       `json:"errors"`
	StartedAt   time.Time `json:"startedAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type TopicStressReceiverStatus struct {
	Active      bool      `json:"active"`
	Topic       string    `json:"topic"`
	RunID       string    `json:"runId"`
	Expected    int       `json:"expected"`
	PayloadSize int       `json:"payloadSize"`
	Rx          int       `json:"rx"`
	Unique      int       `json:"unique"`
	Dup         int       `json:"dup"`
	Corrupt     int       `json:"corrupt"`
	Invalid     int       `json:"invalid"`
	OutOfOrder  int       `json:"outOfOrder"`
	LastSeq     int       `json:"lastSeq"`
	StartedAt   time.Time `json:"startedAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
