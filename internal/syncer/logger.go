package syncer

import (
	"fmt"
	"sync"
	"time"
)

// LogLevel 日志等级/类别
type LogLevel string

const (
	LevelInfo    LogLevel = "INFO"
	LevelSuccess LogLevel = "SUCCESS"
	LevelWarn    LogLevel = "WARN"
	LevelError   LogLevel = "ERROR"
	LevelUpload  LogLevel = "UPLOAD"
	LevelDelete  LogLevel = "DELETE"
)

// LogEntry 单条日志结构
type LogEntry struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Level     LogLevel  `json:"level"`
	Message   string    `json:"message"`
	Details   string    `json:"details,omitempty"`
}

// EventLogger 内存环形日志队列与广播订阅器
type EventLogger struct {
	mu          sync.RWMutex
	entries     []LogEntry
	maxEntries  int
	counter     int64
	subscribers map[chan LogEntry]struct{}
}

// NewEventLogger 创建日志记录器
func NewEventLogger(maxEntries int) *EventLogger {
	if maxEntries <= 0 {
		maxEntries = 1000
	}
	return &EventLogger{
		maxEntries:  maxEntries,
		entries:     make([]LogEntry, 0, maxEntries),
		subscribers: make(map[chan LogEntry]struct{}),
	}
}

// Log 记录一条新日志并广播给订阅者
func (l *EventLogger) Log(level LogLevel, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   msg,
	}

	l.mu.Lock()
	l.counter++
	entry.ID = l.counter

	if len(l.entries) >= l.maxEntries {
		l.entries = l.entries[1:]
	}
	l.entries = append(l.entries, entry)

	// 拷贝当前订阅通道避免广播时持有写锁
	subs := make([]chan LogEntry, 0, len(l.subscribers))
	for ch := range l.subscribers {
		subs = append(subs, ch)
	}
	l.mu.Unlock()

	// 异步非阻塞推送到订阅者
	for _, ch := range subs {
		select {
		case ch <- entry:
		default:
			// 若订阅者通道拥堵，丢弃该事件以保证主线程不被阻塞
		}
	}
}

// Info 记录普通信息
func (l *EventLogger) Info(format string, args ...interface{}) {
	l.Log(LevelInfo, format, args...)
}

// Success 记录成功信息
func (l *EventLogger) Success(format string, args ...interface{}) {
	l.Log(LevelSuccess, format, args...)
}

// Warn 记录告警信息
func (l *EventLogger) Warn(format string, args ...interface{}) {
	l.Log(LevelWarn, format, args...)
}

// Error 记录错误信息
func (l *EventLogger) Error(format string, args ...interface{}) {
	l.Log(LevelError, format, args...)
}

// Upload 记录上传/同步事件
func (l *EventLogger) Upload(format string, args ...interface{}) {
	l.Log(LevelUpload, format, args...)
}

// Delete 记录删除事件
func (l *EventLogger) Delete(format string, args ...interface{}) {
	l.Log(LevelDelete, format, args...)
}

// GetRecent 获取最近日志
func (l *EventLogger) GetRecent(limit int) []LogEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if limit <= 0 || limit > len(l.entries) {
		limit = len(l.entries)
	}
	start := len(l.entries) - limit
	result := make([]LogEntry, limit)
	copy(result, l.entries[start:])
	return result
}

// Clear 清空当前内存日志
func (l *EventLogger) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = make([]LogEntry, 0, l.maxEntries)
}

// Subscribe 注册 SSE 订阅通道
func (l *EventLogger) Subscribe() chan LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()

	ch := make(chan LogEntry, 100)
	l.subscribers[ch] = struct{}{}
	return ch
}

// Unsubscribe 注销订阅通道
func (l *EventLogger) Unsubscribe(ch chan LogEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.subscribers, ch)
	close(ch)
}
