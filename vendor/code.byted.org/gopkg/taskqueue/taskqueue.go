package taskqueue

import (
	"errors"
	"time"
)

// ErrBusy .
var ErrBusy = errors.New("busy")

// Task .
type Task func(req interface{}) (resp interface{}, err error)

// Result .
type Result struct {
	TimeCost time.Duration
	Req      interface{}
	Resp     interface{}
	Error    error
	Done     chan struct{}
}

func newResult(req interface{}) *Result {
	return &Result{
		Req:  req,
		Done: make(chan struct{}),
	}
}

type taskWithResult struct {
	task   Task
	result *Result
}

// TaskQueue .
type TaskQueue struct {
	taskBuffer chan *taskWithResult
	maxConcur  int
	exit       chan struct{}
}

// NewTaskQueue .
func NewTaskQueue(maxConcur, bufSize int) *TaskQueue {
	return &TaskQueue{
		maxConcur:  maxConcur,
		taskBuffer: make(chan *taskWithResult, bufSize),
	}
}

// Submit .
func (q *TaskQueue) Submit(t Task, req interface{}) (*Result, error) {
	result := newResult(req)
	tr := &taskWithResult{t, result}
	select {
	case q.taskBuffer <- tr:
		return result, nil
	default:
	}
	return nil, ErrBusy
}

// Start .
func (q *TaskQueue) Start() {
	q.exit = make(chan struct{})
	for i := 0; i < q.maxConcur; i++ {
		go q.tasker()
	}
}

// Stop .
func (q *TaskQueue) Stop() {
	close(q.exit)
}

func (q *TaskQueue) tasker() {
	for {
		select {
		case <-q.exit:
			return
		case t := <-q.taskBuffer:
			begin := time.Now()
			t.result.Resp, t.result.Error = t.task(t.result.Req)
			t.result.TimeCost = time.Now().Sub(begin)
			close(t.result.Done)
		}
	}
}
