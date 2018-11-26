package detector

import (
	"sync"
	"fmt"
	"errors"
	"context"
	"code.byted.org/microservice/tsad/utils"
	"time"
)

var singleton *detector
var lock sync.Mutex

func Start(op *Options) error {
	lock.Lock()
	defer lock.Unlock()
	if singleton != nil {
		return fmt.Errorf("has alread started a detector")
	}

	if op.MaxTasks == 0 {
		return errors.New("no MaxConTasks")
	}
	if op.P == nil {
		return fmt.Errorf("no Plugins")
	}
	if err := op.P.Valid(); err != nil {
		return err
	}
	if op.TaskLeaser == nil {
		return fmt.Errorf("no TaskLeaser")
	}

	singleton = &detector{
		O:         op,
		status:    _STATUS_INIT,
		tasks:     make(map[string]*Task),
		contexts:  make(map[string]context.Context),
		cancels:   make(map[string]func()),
		exit:      make(chan struct{}),
		logger:    utils.NewLogger("detector"),
		metricser: utils.NewDefaultMetricser(),
	}
	return singleton.start()
}

func ForecastInterval(name string, stamps []time.Time) ([]*ForecastTS, error) {
	return singleton.ForecastInterval(name, stamps)
}

func TaskInfo(name string) (*Task, bool) {
	return singleton.TaskInfo(name)
}

func SubmitTask(t TaskMeta) error {
	return singleton.SubmitTask(t)
}

func CancelTask(name string) error {
	return singleton.CancelTask(name)
}

func AllTasks() map[string]*Task {
	return singleton.AllTasks()
}
