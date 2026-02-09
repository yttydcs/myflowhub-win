package logs

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yttydcs/myflowhub-core/eventbus"
)

const EventLogLine = "logs.line"

type LogLine struct {
	Level            string    `json:"level"`
	Message          string    `json:"message"`
	Time             time.Time `json:"time"`
	Payload          []byte    `json:"payload,omitempty"`
	PayloadLen       int       `json:"payload_len,omitempty"`
	PayloadTruncated bool      `json:"payload_truncated,omitempty"`
}

type LogService struct {
	mu       sync.Mutex
	lines    []LogLine
	maxLines int
	paused   atomic.Bool
	bus      eventbus.IBus
}

func New(bus eventbus.IBus, maxLines int) *LogService {
	if maxLines <= 0 {
		maxLines = 2000
	}
	return &LogService{bus: bus, maxLines: maxLines}
}

func (s *LogService) Pause(paused bool) {
	s.paused.Store(paused)
}

func (s *LogService) IsPaused() bool {
	return s.paused.Load()
}

func (s *LogService) Append(level, message string) {
	if s == nil || s.IsPaused() {
		return
	}
	s.appendLine(LogLine{Level: level, Message: message, Time: time.Now()})
}

func (s *LogService) Appendf(level, format string, args ...any) {
	if s == nil {
		return
	}
	s.Append(level, fmt.Sprintf(format, args...))
}

func (s *LogService) AppendPayload(level, message string, payload []byte, payloadLen int, truncated bool) {
	if s == nil || s.IsPaused() {
		return
	}
	line := LogLine{
		Level:            level,
		Message:          message,
		Time:             time.Now(),
		Payload:          payload,
		PayloadLen:       payloadLen,
		PayloadTruncated: truncated,
	}
	s.appendLine(line)
}

func (s *LogService) appendLine(line LogLine) {
	s.mu.Lock()
	s.lines = append(s.lines, line)
	if len(s.lines) > s.maxLines {
		s.lines = s.lines[len(s.lines)-s.maxLines:]
	}
	s.mu.Unlock()
	if s.bus != nil {
		_ = s.bus.Publish(context.Background(), EventLogLine, line, nil)
	}
}

func (s *LogService) Lines() []LogLine {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]LogLine, len(s.lines))
	copy(out, s.lines)
	return out
}
