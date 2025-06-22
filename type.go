package cronJob

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
	logger   *Logger
	mutex    sync.Mutex
	wait     sync.WaitGroup
	heap     taskHeap
	chain    taskChain
	parser   parser
	stop     chan struct{}
	add      chan *task
	remove   chan int
	location *time.Location
	next     int
	running  bool
}

type task struct {
	id       int
	schedule schedule
	action   func()
	next     time.Time
	prev     time.Time
	enable   bool
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
