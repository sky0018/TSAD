package detector

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"code.byted.org/microservice/tsad/worker/ts"
	"errors"
	"code.byted.org/microservice/tsad/utils"
)

const (
	_DefaultTaskLeaseDuration = time.Minute * 30

	_STATUS_INIT    = iota
	_STATUS_STARTED
	_STATUS_STOPPED
)

// Options .
type Options struct {
	MaxTasks int

	P          *Plugins
	TaskLeaser TaskLeaser
}

type detector struct {
	O *Options

	// runtime fields
	tasks    map[string]*Task
	contexts map[string]context.Context
	cancels  map[string]func()
	status   int
	exit     chan struct{}
	lock     sync.RWMutex

	// dependency
	logger    utils.Logger
	metricser utils.Metricser
}

func (d *detector) start() error {
	d.lock.RLock()
	defer d.lock.RUnlock()

	if d.status != _STATUS_INIT {
		return errors.New("detector has already been started")
	}
	d.status = _STATUS_STARTED

	go d.heartbeat()
	return nil
}

func (d *detector) heartbeat() {
	for range time.Tick(time.Second * 30) {
		counters := d.TaskCounter()
		running := counters[TaskInit] + counters[TaskDerive] + counters[TaskProcess]

		if err := d.O.P.Heartbeat(running); err != nil {
			d.logger.Errorf("heartbeat UpdateInfo err: %v", err)
		} else {
			d.logger.Infof("heartbeat success")
		}

		for s, c := range counters {
			d.metricser.EmitStore("detector.task.counter", c, map[string]string{"state": string(s)})
			d.logger.Infof("task counter, state=%v, number=%v", string(s), c)
		}

		tasks := d.AllTasks()
		tsCounter := make(map[TSState]int)
		for _, t := range tasks {
			for _, s := range t.TSs() {
				tsCounter[s.State()] ++
			}
		}

		for s, c := range tsCounter {
			d.metricser.EmitStore("detector.ts.counter", c, map[string]string{"state": string(s)})
			d.logger.Infof("time-series counter, state=%v, number=%v", string(s), c)
		}
	}
}

func (d *detector) stop() error {
	d.lock.Lock()
	defer d.lock.Unlock()

	if d.status == _STATUS_INIT {
		return errors.New("detector has not been started yet")
	}

	if d.status == _STATUS_STOPPED {
		return errors.New("detector has already been stopped")
	}

	return nil
}

func (d *detector) addTask(t *Task) {
	d.lock.Lock()
	defer d.lock.Unlock()
	ctx, cancel := context.WithCancel(context.Background())
	d.tasks[t.Name] = t
	d.contexts[t.Name] = ctx
	d.cancels[t.Name] = cancel
}

func (d *detector) cancelTask(t *Task) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.cancels[t.Name]()
	t.SetState(TaskCancel)
}

func (d *detector) taskDoneCh(t *Task) <-chan struct{} {
	d.lock.RLock()
	defer d.lock.RUnlock()
	return d.contexts[t.Name].Done()
}

func (d *detector) taskHasDone(t *Task) bool {
	d.lock.RLock()
	defer d.lock.RUnlock()
	select {
	case <-d.contexts[t.Name].Done():
		return true
	default:
		return false
	}
}

func (d *detector) renewal(t *Task) {
	base := 6
	consecErr := 0
	tick := time.Tick(_DefaultTaskLeaseDuration / time.Duration(base))

	for {
		select {
		case <-d.taskDoneCh(t):
			return
		case <-tick:
			err := d.O.TaskLeaser.Renewal(t.Name, _DefaultTaskLeaseDuration)
			if err != nil {
				d.logger.Errorf("renewal task %v err: %v", t.Name, err)
				consecErr++
			} else {
				consecErr = 0
			}
			if consecErr == base-1 { // cancel this task
				d.cancelTask(t)
				d.logger.Errorf("renewal task=%v, err=%v", t.Name, err)
				t.SetErr(fmt.Errorf("renewal err=%v", err))
				return
			}
		}
	}
}

func (d *detector) submitTask(t *Task) {
	d.logger.Infof("receive a task=%v", t.Name)

	t.SetState(TaskInit)
	d.addTask(t)
	defer d.cancelTask(t)

	if err := d.O.TaskLeaser.Lease(t.Name, _DefaultTaskLeaseDuration); err != nil {
		d.logger.Errorf("lease task=%v err=%v", t.Name, err)
		t.SetErr(fmt.Errorf("lease task err=%v", err))
		return
	}

	// use a goroutine to renewal this task periodically
	go d.renewal(t)

	// derive time-series from this task
	srcs, err := d.O.P.DeriveSource(t.DataSource)
	if err != nil {
		d.logger.Errorf("derive task=%v err=%v", t.Name, err)
		d.metricser.EmitCounter("detector.derive_err", 1, nil)
		t.SetErr(fmt.Errorf("derive err=%v", err))
		return
	}

	if len(srcs) == 0 {
		d.logger.Errorf("task=%v, err=no data source can be derived from %v", t.Name, t.DataSource)
		d.metricser.EmitCounter("detector.derive_zero", 1, nil)
		t.SetErr(fmt.Errorf("no datasource can be derived from %v", t.DataSource))
		return
	}

	t.SetState(TaskDerive)

	for _, src := range srcs {
		s := newTimeSeries(t.Name, src)
		t.AddTS(s)
		go d.process(t, s)
	}

	t.SetState(TaskProcess)
	<-d.taskDoneCh(t)
}

func (d *detector) process(t *Task, s *TimeSeries) {
	retryInterval := time.Minute * 10
	for {
		s.Reset()
		if normal := d.processTS(t, s); normal {
			// normal exit for retrain the model
			continue
		}

		// abnormal exit because there are some errors happened
		s.SetState(TSError)

		if d.taskHasDone(t) {
			s.SetState(TSCancel)
			return
		}

		retryInterval += retryInterval / 2
		if retryInterval >= time.Hour*6 {
			retryInterval = time.Hour * 6
		}
		time.Sleep(retryInterval)
	}
}

func (d *detector) processTS(t *Task, s *TimeSeries) (normal bool) {
	// recover model from stored data
	recovered := false
	var model TSModel
	if name, data, stamp, err := d.O.P.ReadModelData(s.DataSource); err == nil {
		expiration := time.Hour * 24
		if stamp.Add(expiration).After(time.Now()) {
			model, err = d.O.P.RecoverModel(name, []byte(data))
			if err == nil {
				recovered = true
				d.metricser.EmitCounter("detector.recover.succ", 1, nil)
			} else {
				d.logger.Errorf("ts=%v, recover model %v err: %v", t.Name, s.Name(), name, err)
				d.metricser.EmitCounter("detector.recover.err", 1, nil)
			}
		}
	} else {
		d.logger.Errorf("ts=%v, read model data err: %v", t.Name, s.Name(), err)
		d.metricser.EmitCounter("detector.recover.read_err", 1, nil)
	}

	if recovered {
		s.SetState(TSRecoverSucc)
		s.SetModel(model)
	} else {
		s.SetState(TSRecoverErr)

		// training now, fetch a long-time data to train model
		if d.taskHasDone(t) {
			return false
		}
		end := time.Now()
		trainingDataLength := getInt(t.Configs, "training_data_length", 3) // day
		begin := end.Add(-time.Hour * 24 * time.Duration(trainingDataLength))
		tsData, err := d.O.P.FetchFromTo(context.Background(), s, begin, end)
		if err != nil {
			d.logger.Errorf("ts=%v, fetch latest data err=%v", s.Name(), err)
			d.metricser.EmitCounter("detector.fetch.err", 1, nil)
			s.SetErr(fmt.Errorf("fetch latest data err=%v", err))
			return false
		}

		s.SetState(TSFetch)
		if d.taskHasDone(t) {
			return false
		}

		// clean this time-series
		tsData, err = d.O.P.Preprocess(tsData)
		if err != nil {
			d.logger.Errorf("ts=%v preprocess data err=%v", s.Name(), err)
			d.metricser.EmitCounter("detector.preprocess.err", 1, nil)
			s.SetErr(fmt.Errorf("preprocess data err=%v", err))
			return false
		}

		s.SetState(TSPreprocess)
		if d.taskHasDone(t) {
			return false
		}

		// train model
		model, err = d.O.P.Train(tsData, d.O.P.ModelAdapter)
		if err != nil {
			d.logger.Errorf("ts=%v, train model err=%v", s.Name(), err)
			d.metricser.EmitCounter("detector.train.err", 1, nil)
			s.SetErr(fmt.Errorf("train model err=%v", err))
			return false
		}

		s.SetState(TSTrain)
		s.SetModel(model)
		if d.taskHasDone(t) {
			return false
		}

		// store this model
		data, err := model.ModelData()
		if err != nil {
			d.logger.Errorf("ts=%v, model %v data err: %v", s.Name(), model.Name(), err)
		} else {
			if err := d.O.P.StoreModelData(s.DataSource, model.Name(), string(data), time.Now()); err != nil {
				d.logger.Errorf("ts=%v, store model(%v) data err: %v", s.Name(), model.Name(), err)
			}
		}
	}

	if d.taskHasDone(t) {
		return false
	}

	d.metricser.EmitCounter("detector.process.succ", 1, nil)
	return d.monitor(t, s)
}

func (d *detector) monitor(t *Task, s *TimeSeries) (normal bool) {
	s.SetState(TSMonitor)
	m := s.Model()

	beginAt := time.Now()
	min := time.Hour * 24 // half day
	max := time.Hour * 36 // one day
	retrain := time.Duration(rand.Intn(int(max-min))) + min

	checkFreqMin := getInt(t.Configs, "check_freq_min", 5)
	consAlert := 0
	for range time.Tick(time.Minute * time.Duration(checkFreqMin)) {
		if d.taskHasDone(t) {
			return false
		}

		if time.Now().Sub(beginAt) > retrain {
			return true // let it retrain
		}

		checkDataMin := getInt(t.Configs, "check_data_min", 8)
		latestData, err := d.O.P.FetchFromTo(context.Background(), s, time.Now().Add(-time.Minute*time.Duration(checkDataMin)), time.Now())
		if err != nil {
			d.logger.Errorf("ts=%v, fetch latest data when monitor err=%v", s.Name(), err)
			continue
		}

		if latestData.N() == 0 {
			d.logger.Infof("ts=%v, latest data if empty", s.Name())
			continue
		}

		badPoints := 0
		var latestBad ts.Point
		alertSensitive := getFloat64(t.Configs, "alert_sensitive", 0.5)
		for _, p := range latestData.Points() {
			lower, upper := m.ForecastInterval(p.Stamp())
			if (p.Value() > upper+math.Abs(upper*alertSensitive) &&
				p.Value()-upper > 10) ||
				(p.Value() < lower-math.Abs(lower*alertSensitive) &&
					lower-p.Value() > 10) {
				badPoints++
				latestBad = p
			}
		}

		if badPoints == latestData.N() {
			lower, upper := m.ForecastInterval(latestBad.Stamp())
			d.O.P.Alert(t, s, lower, upper, latestBad)
			consAlert++
		} else {
			consAlert = 0
		}

		if consAlert * checkFreqMin > 15 {
			return true // 如果长时间异常, 我们认为是模型数据不够充分, 自动重新训练
		}
	}

	// assert(false)
	return false
}

func (d *detector) ForecastInterval(name string, stamps []time.Time) ([]*ForecastTS, error) {
	if len(stamps) == 0 {
		return nil, fmt.Errorf("no stamps")
	}

	task, ok := d.TaskInfo(name)
	if !ok {
		return nil, fmt.Errorf("no such a task: %s", name)
	}
	if task.State() != TaskProcess {
		return nil, fmt.Errorf("not in process state(now=%v), please try later", task.State())
	}

	tss := task.TSs()
	forecastTSs := make([]*ForecastTS, 0, len(tss))
	for _, t := range tss {
		forecastTS := &ForecastTS{
			DataSource: t.DataSource,
		}

		err, stamp := t.Err()
		if err != nil {
			forecastTS.Error = fmt.Errorf("time series has err=%v, stamp=%v", err, stamp)
			forecastTSs = append(forecastTSs, forecastTS)
			continue
		}

		if t.Model() == nil {
			forecastTS.Error = fmt.Errorf("state=%v, has no model now, please try later", t.State())
			forecastTSs = append(forecastTSs, forecastTS)
			continue
		}

		ctx := context.WithValue(context.Background(), "noblock", true)
		observeTS, err := d.O.P.FetchFromTo(ctx, t, stamps[0], stamps[len(stamps)-1])
		if err != nil {
			forecastTS.Error = fmt.Errorf("please try later, fetch %v err=%v", t.DataSource, err)
			forecastTSs = append(forecastTSs, forecastTS)
			continue
		}
		observe := make([]*Point, 0, observeTS.N())
		for _, p := range observeTS.Points() {
			observe = append(observe, &Point{
				Value: p.Value(),
				Stamp: p.Stamp(),
			})
		}

		upper := make([]*Point, 0, len(stamps))
		lower := make([]*Point, 0, len(stamps))
		for _, stamp := range stamps {
			l, u := t.Model().ForecastInterval(stamp)
			lower = append(lower, &Point{
				Stamp: stamp,
				Value: l,
			})
			upper = append(upper, &Point{
				Stamp: stamp,
				Value: u,
			})
		}

		forecastTS.Observe = observe
		forecastTS.Uppert = upper
		forecastTS.Lower = lower
		forecastTSs = append(forecastTSs, forecastTS)
	}

	return forecastTSs, nil
}

func (d *detector) CancelTask(name string) error {
	t, ok := d.TaskInfo(name)
	if !ok {
		return fmt.Errorf("task %v not found", name)
	}
	d.cancelTask(t)
	return nil
}

func (d *detector) SubmitTask(t TaskMeta) error {
	d.lock.Lock()
	defer d.lock.Unlock()

	totalRunning := 0
	for _, t := range d.tasks {
		if t.State() != TaskCancel {
			totalRunning ++
		}
	}
	if totalRunning > d.O.MaxTasks {
		return fmt.Errorf("there are too many tasks")
	}

	task, err := newTask(t)
	if err != nil {
		return fmt.Errorf("create task err=%v", err)
	}

	go d.submitTask(task)
	return nil
}

func (d *detector) TaskInfo(name string) (*Task, bool) {
	d.lock.RLock()
	defer d.lock.RUnlock()
	t, ok := d.tasks[name]
	return t, ok
}

func (d *detector) TaskCounter() map[TaskState]int {
	d.lock.RLock()
	defer d.lock.RUnlock()
	results := make(map[TaskState]int)
	for _, t := range d.tasks {
		results[t.State()] ++
	}
	return results
}

func (d *detector) AllTasks() map[string]*Task {
	d.lock.RLock()
	defer d.lock.RUnlock()
	results := make(map[string]*Task, len(d.tasks))
	for name, t := range d.tasks {
		results[name] = t
	}
	return results
}
