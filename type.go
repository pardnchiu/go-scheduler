package goCron

import (
	"sync"
	"time"

	goLogger "github.com/pardnchiu/go-logger"
)

const (
	defaultLogPath      = "./logs/cron.log"
	defaultLogMaxSize   = 16 * 1024 * 1024
	defaultLogMaxBackup = 5
)

type Log = goLogger.Log
type Logger = goLogger.Logger

type Config struct {
	Log      *Log
	Location *time.Location
}

type cron struct {
	logger    *Logger
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
	Delay       time.Duration
	OnDelay     func()
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
