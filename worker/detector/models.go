package detector

import (
	"encoding/json"
	"fmt"
	"time"

	"code.byted.org/collect/grass/pkg/util"
	"sync"
)

type DataSourceType string

const (
	DataSourceTypeTSDB = "tsdb"
)

type DataSource struct {
	Type  DataSourceType `json:"type"`
	Key   string         `json:"key"`
	Extra string         `json:"extra"`
}

func (ds DataSource) String() string {
	return ds.JSON()
}

func (ds DataSource) JSON() string {
	buf, _ := json.Marshal(ds)
	return string(buf)
}

type TaskState string

const (
	TaskInit    TaskState = "init"
	TaskDerive  TaskState = "derive"
	TaskProcess TaskState = "process"
	TaskCancel  TaskState = "cancel"
)

type TaskMeta struct {
	Name       string     `json:"name"` // primary key
	DataSource DataSource `json:"data_source"`
	Config     string     `json:"config"`
}

type TaskRuntime struct {
	// these fields are created when init and immutable
	Configs map[string]interface{}

	// these fields protected by lock
	state    TaskState
	err      error
	errStamp time.Time
	tss      map[string]*TimeSeries
	lock     sync.RWMutex
}

func (tr *TaskRuntime) State() TaskState {
	tr.lock.RLock()
	defer tr.lock.RUnlock()
	return tr.state
}

func (tr *TaskRuntime) SetState(state TaskState) {
	tr.lock.Lock()
	defer tr.lock.Unlock()
	tr.state = state
}

func (tr *TaskRuntime) Err() (error, time.Time) {
	tr.lock.RLock()
	defer tr.lock.RUnlock()
	return tr.err, tr.errStamp
}

func (tr *TaskRuntime) SetErr(err error) {
	tr.lock.Lock()
	defer tr.lock.Unlock()
	tr.err = err
	tr.errStamp = time.Now()
}

func (tr *TaskRuntime) AddTS(s *TimeSeries) {
	tr.lock.Lock()
	defer tr.lock.Unlock()
	tr.tss[s.Name()] = s
}

func (tr *TaskRuntime) TSs() map[string]*TimeSeries {
	tr.lock.RLock()
	defer tr.lock.RUnlock()
	tss := make(map[string]*TimeSeries, len(tr.tss))
	for name, s := range tr.tss {
		tss[name] = s
	}
	return tss
}

type Task struct {
	TaskMeta
	TaskRuntime
}

func newTask(meta TaskMeta) (*Task, error) {
	confMap := make(map[string]interface{})
	if meta.Config != "" {
		if err := json.Unmarshal([]byte(meta.Config), confMap); err != nil {
			return nil, fmt.Errorf("invalid config")
		}
	}

	return &Task{
		TaskMeta: meta,
		TaskRuntime: TaskRuntime{
			Configs: confMap,
			state:   TaskInit,
			tss:     make(map[string]*TimeSeries),
		},
	}, nil
}

func (t *Task) MarshalJSON() ([]byte, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	jMap := map[string]interface{}{
		"name":             t.Name,
		"data_source":      t.DataSource,
		"config":           t.Configs,
		"state":            t.State,
		"last_error":       t.err,
		"last_error_stamp": t.errStamp,
		"ts_states":        t.tss,
	}

	return json.Marshal(jMap)
}

type TSState string

const (
	TSInit        = "init"
	TSRecoverSucc = "recover_succ"
	TSRecoverErr  = "recover_err"
	TSFetch       = "fetch"
	TSPreprocess  = "preprocess"
	TSTrain       = "train"
	TSMonitor     = "monitor"
	TSError       = "error"
	TSCancel      = "cancel"
)

// TimeSeries 一个task或许会产生多个ts, ts在运行时由task产生, 不需要持久化存储;
type TimeSeries struct {
	TaskName    string     // the task which the ts derived from
	DataSource  DataSource // data source for this ts
	DerivedAt   time.Time
	DerivedHost string

	// runtime information
	state      TSState
	model      TSModel
	err        error
	errStamp   time.Time
	detectedAt time.Time
	lock       sync.RWMutex
}

func (ts *TimeSeries) State() TSState {
	ts.lock.RLock()
	defer ts.lock.RUnlock()
	return ts.state
}

func (ts *TimeSeries) SetState(state TSState) {
	ts.lock.Lock()
	defer ts.lock.Unlock()
	ts.state = state
}

func (ts *TimeSeries) SetModel(model TSModel) {
	ts.lock.Lock()
	defer ts.lock.Unlock()
	ts.model = model
}

func (ts *TimeSeries) Model() TSModel {
	ts.lock.RLock()
	defer ts.lock.RUnlock()
	return ts.model
}

func (ts *TimeSeries) SetErr(err error) {
	ts.lock.Lock()
	defer ts.lock.Unlock()
	ts.err = err
	ts.errStamp = time.Now()
}

func (ts *TimeSeries) Err() (error, time.Time) {
	ts.lock.RLock()
	defer ts.lock.RUnlock()
	return ts.err, ts.errStamp
}

func (ts *TimeSeries) Reset() {
	ts.lock.Lock()
	defer ts.lock.Unlock()
	ts.state = TSInit
	ts.model = nil
	ts.err = nil
	ts.errStamp = time.Unix(0, 0)
}

func (ts *TimeSeries) Clear() {
	ts.lock.Lock()
	defer ts.lock.Unlock()
	ts.model = nil // to save memory
}

func (ts *TimeSeries) MarshalJSON() ([]byte, error) {
	ts.lock.RLock()
	defer ts.lock.RUnlock()
	return json.Marshal(map[string]interface{}{
		"task_name":        ts.TaskName,
		"data_source":      ts.DataSource,
		"derived_at":       ts.DerivedAt,
		"state":            ts.State,
		"model":            ts.model.Name(),
		"last_error":       ts.err,
		"last_error_stamp": ts.errStamp,
		"last_detect_at":   ts.detectedAt,
	})
}

func (ts *TimeSeries) Name() string {
	return fmt.Sprintf("%s:%s", ts.TaskName, ts.DataSource)
}

func newTimeSeries(name string, src DataSource) *TimeSeries {
	host, _ := util.LocalIP()
	return &TimeSeries{
		TaskName:    name,
		DataSource:  src,
		DerivedAt:   time.Now(),
		DerivedHost: host,
		state:       TSInit,
	}
}

type Point struct {
	Value float64   `json:"value"`
	Stamp time.Time `json:"stamp"`
}

type ForecastTS struct {
	DataSource DataSource `json:"data_source"`
	Error      error      `json:"error"`
	Observe    []*Point   `json:"observe"`
	Uppert     []*Point   `json:"upper"`
	Lower      []*Point   `json:"lower"`
}

func (f *ForecastTS) MarshalJSON() ([]byte, error) {
	err := ""
	if f.Error != nil {
		err = f.Error.Error()
	}

	jMap := map[string]interface{}{
		"data_source": f.DataSource,
		"observe":     f.Observe,
		"upper":       f.Uppert,
		"lower":       f.Lower,
		"error":       err,
	}

	return json.Marshal(jMap)
}
