package cron

import (
	"log"
	"sync"
	"time"
)

type Logger interface {
	Info(format string, args ...any)
	Error(format string, args ...any)
	Debug(format string, args ...any)
}

type StandardLogger struct {
	*log.Logger
}

type NoOpLogger struct{}

type Config struct {
	Logger   Logger
	Location *time.Location
}

type cron struct {
	logger    Logger
	mutex     sync.Mutex
	wait      sync.WaitGroup
	heap      taskHeap
	chain     taskChain
	parser    parser
	stop      chan struct{}
	add       chan *task
	remove    chan int64
	removeAll chan struct{}
	location  *time.Location
	next      int64
	running   bool
}

type task struct {
	ID          int64
	Description string
	Schedule    schedule
	Action      func()
	Next        time.Time
	Prev        time.Time
	Enable      bool
}

type schedule interface {
	next(time.Time) time.Time
}

type scheduleResult struct {
	minute,
	hour,
	dom,
	month,
	dow scheduleField
}

type scheduleField struct {
	Value int  // 具體數值
	All   bool // 是否匹配所有值（對應 "*"）
	Step  int  // 步長值（對應 "*/n"）
}

type delayScheduleResult struct {
	delay time.Duration
}

type taskHeap []*task
type taskChain []func(func()) func()

type parser struct{}
