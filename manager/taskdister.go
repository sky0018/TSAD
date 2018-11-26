package manager

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"
	"code.byted.org/microservice/tsad/utils"
)

func startTaskDister() {
	state = _TDStateNotOnduty
	exit = make(chan struct{})
	go start()
}

const (
	_TDStateStopped   = iota // internal variables
	_TDStateOnduty
	_TDStateNotOnduty

	_TDDefaultLeaseDuration = time.Second * 60
)

var (
	state      uint64
	exit       chan struct{}
	distLogger = utils.NewLogger("taskdister")
)

func stopTaskDister() error {
	atomic.StoreUint64(&state, _TDStateStopped)
	close(exit)
	return nil
}

func start() {
	var exitAsync chan struct{}
	sleepInterval := 20 * time.Second
	for {
		stateNow := atomic.LoadUint64(&state)

		switch stateNow {
		case _TDStateStopped:
			return
		case _TDStateNotOnduty: // try to lease
			err := Lease(_TDDefaultLeaseDuration)
			if err == nil { // succ
				exitAsync = make(chan struct{})
				go distributeAsync(exitAsync)
				atomic.StoreUint64(&state, _TDStateOnduty)
			}

			distLogger.Infof("[taskdister] lease result=%v", err)

		case _TDStateOnduty: // try to renewal
			err := Renewal(_TDDefaultLeaseDuration)
			if err != nil {
				distLogger.Warnf("[taskdister] renewal duty err=%v", err)
				close(exitAsync)
				atomic.StoreUint64(&state, _TDStateNotOnduty)
			}

			distLogger.Infof("[taskdister] renewal result=%v", err)
		}

		time.Sleep(sleepInterval)
	}
}

func distributeAsync(localExit chan struct{}) {
	distLogger.Infof("[taskdister] distributeAsync")
	// first
	distributeTask()

	tick := time.Tick(time.Second * 120)
	for {
		select {
		case <-exit:
			return
		case <-localExit:
			return
		case <-tick:
			distributeTask()
			distLogger.Infof("[taskdister] do distributeAsync")
		}
	}
}

func distributeTask() {
	distLogger.Infof("[taskdister] start distributeTask")
	utils.EmitStore("dist_task", 1, nil)

	tasks, err := GetTasks()
	if err != nil {
		distLogger.Errorf("[taskdister] get tasks err=%v", err)
		return
	}
	alives := make([]*Task, 0, len(tasks))
	for _, t := range tasks {
		if t.State != TaskStopped {
			alives = append(alives, t)
		}
	}
	tasks = alives

	distLogger.Infof("[taskdister] total tasks=%v", len(tasks))

	now := time.Now()
	expTasks := make([]*Task, 0, len(tasks))
	for _, t := range tasks {
		if t.LockExpiration.After(now) {
			continue
		}
		expTasks = append(expTasks, t)
	}

	distLogger.Infof("[taskdister] number of tasks to distribute=%v", len(expTasks))

	if len(expTasks) == 0 {
		distLogger.Infof("[taskdister] no task can be distribute")
		return
	}

	detectors, err := GetAliveDetectors()
	if err != nil {
		distLogger.Errorf("[taskdister] get detector infos err=%v", err)
		return
	}
	distLogger.Infof("[taskdister] number of detectors=%v", len(detectors))
	if len(detectors) == 0 {
		distLogger.Errorf("[taskdister] no alive detector")
		return
	}

	// find a balance number of task for all detectors
	bn := findBalanceNumber(detectors, len(expTasks))
	distLogger.Infof("[taskdister] balance number=%v", bn)

	// add tasks to each detector
	for _, d := range detectors {
		if len(expTasks) == 0 {
			break
		}
		if d.NumTasks >= bn {
			continue
		}

		count := bn - d.NumTasks
		if len(expTasks) < count {
			count = len(expTasks)
		}

		err := SubmitTasksToDetector(d, expTasks[:count])
		if err != nil {
			distLogger.Errorf("[taskdister] submit to %v err=%v", d, err)
		}

		expTasks = expTasks[count:]
	}
}

// findBalanceNumber use binary search to find the number of task each detector should have
func findBalanceNumber(detectors []*Detector, toAdd int) int {
	total := toAdd
	for _, d := range detectors {
		total += d.NumTasks
	}

	left := 0
	right := total + 1

	for left < right {
		mid := (left + right) / 2
		idles := 0
		for _, d := range detectors {
			if d.NumTasks < mid {
				idles += mid - d.NumTasks
			}
		}

		if idles >= toAdd {
			right = mid
		} else {
			left = mid + 1
		}
	}

	return left
}

func isAlive(d *Detector) bool {
	return d.HeartBeat.Add(time.Second * 120).After(time.Now())
}

// GetAliveDetectors .
func GetAliveDetectors() ([]*Detector, error) {
	ds, err := GetDetectors()
	if err != nil {
		return nil, err
	}

	alives := make([]*Detector, 0, len(ds))
	for _, d := range ds {
		if isAlive(d) {
			alives = append(alives, d)
		}
	}

	return alives, nil
}

// SubmitTasksToDetector .
func SubmitTasksToDetector(d *Detector, ts []*Task) error {
	var resp []string
	url := fmt.Sprintf("http://%v:%v/tsad/api/detector/submit_batch_tasks", d.Host, config.WorkerPort)

	metas := make([]TaskMeta, 0, len(ts))
	for _, t := range ts {
		var src DataSource
		if err := json.Unmarshal([]byte(t.DataSource), &src); err != nil {
			return fmt.Errorf("invalid datasource: %v", t.DataSource)
		}
		metas = append(metas, TaskMeta{
			Name:       t.Name,
			DataSource: src,
		})
	}

	if err := PostModel(url, metas, &resp); err != nil {
		return err
	}

	var batchErr error
	for i, r := range resp {
		if r != "ok" {
			batchErr = fmt.Errorf("%v; task %v err: %v", batchErr, ts[i], r)
		}
	}

	return batchErr
}
