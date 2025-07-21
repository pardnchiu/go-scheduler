package goCron

import (
	"log/slog"
	"sync"
	"time"
)

var (
	maxWorker = 2
	// logger    *slog.Logger
)

const (
	TaskPending int = iota
	TaskRunning
	TaskCompleted
	TaskFailed
)

type Config struct {
	Location *time.Location
}

type cron struct {
	mutex     sync.Mutex
	wait      sync.WaitGroup
	heap      taskHeap
	parser    parser
	stop      chan struct{}
	add       chan *task
	remove    chan int64
	removeAll chan struct{}
	location  *time.Location
	depend    *depend
	next      int64
	running   bool
	logger    *slog.Logger
}

type depend struct {
	mutex    sync.RWMutex
	wait     sync.WaitGroup
	manager  *dependManager
	running  bool
	queue    chan Wait
	stopChan chan struct{}
	logger   *slog.Logger
}

type Wait struct {
	ID    int64
	Delay time.Duration
	State WaitState
}

type WaitState int

const (
	Stop WaitState = iota
	Skip
)

type dependManager struct {
	mutex   sync.RWMutex
	list    map[int64]*task
	waiting map[int64][]*task
}

type task struct {
	mutex       sync.RWMutex
	ID          int64
	description string
	schedule    schedule
	action      func() error
	next        time.Time
	prev        time.Time
	enable      bool
	delay       time.Duration
	wait        time.Duration
	waitState   WaitState
	onDelay     func()
	after       []Wait
	state       int
	result      *taskResult
	startChan   chan struct{}
	doneChan    chan taskResult
}

type taskResult struct {
	ID       int64
	status   int
	start    time.Time
	end      time.Time
	duration time.Duration
	error    error
}

type taskState struct {
	done    bool
	waiting []Wait
	failed  *int64
	error   error
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
	Value  int
	Values []int
	All    bool
	Step   int
}

type delayScheduleResult struct {
	delay time.Duration
}

type taskHeap []*task
type parser struct{}
