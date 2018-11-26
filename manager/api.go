package manager

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// UpdateTask update a task by its name
func UpdateTask(c *gin.Context) {
	type req struct {
		OldName    string     `json:"old_name"`
		NewName    string     `json:"new_name"`
		DataSource DataSource `json:"data_source"`
		Config     string     `json:"config"`
	}
	var r req
	if err := c.BindJSON(&r); err != nil {
		c.String(400, "invalid argument")
		return
	}

	t, err := GetTaskByName(r.OldName)
	if err != nil {
		c.String(500, "query task err: %v", err)
		return
	}

	// cancel this task
	if t.State != TaskStopped {
		if err := cancelTask(t); err != nil {
			c.String(500, "cancel this task err: %v", err)
			return
		}
	}

	// update this task
	t.Name = r.NewName
	src, _ := json.Marshal(r.DataSource)
	t.DataSource = string(src)
	t.Config = r.Config
	if err := UpdateTaskByName(r.OldName, t); err != nil {
		c.String(500, "update task err: %v", err)
		return
	}

	c.String(200, "ok")
}

/*
SubmitTask .
	submit a task to TSAD cluster
*/
func SubmitTask(c *gin.Context) {
	var meta TaskMeta
	if err := c.BindJSON(&meta); err != nil {
		c.String(400, "invalid argument")
		return
	}

	src, err := json.Marshal(meta.DataSource)
	if err != nil {
		c.String(400, "marshal datasource err: %v", err)
		return
	}

	// just store this task to db and let taskdister distribute this
	//  task later
	err = InsertTask(meta.Name, string(src), meta.Config)
	if err != nil {
		c.String(500, err.Error())
		return
	}

	c.String(200, "ok")
}

func detectorAllTasks(d *Detector) ([]*TaskDetail, error) {
	url := fmt.Sprintf("http://%v:%v/tsad/api/detector/all_task_detail", d.Host, config.WorkerPort)
	var tasks []*TaskDetail
	if err := GetModel(url, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// AllTaskDetail .
func AllTaskDetail(c *gin.Context) {
	ds, err := GetAliveDetectors()
	if err != nil {
		c.String(500, "get alive detectors err: %v", err)
		return
	}

	ts, err := GetTasks()
	if err != nil {
		c.String(500, "")
		return
	}
	tasks := make(map[string]*TaskDetail, len(ts))
	for _, t := range ts {
		var src DataSource
		json.Unmarshal([]byte(t.DataSource), &src)
		tasks[t.Name] = &TaskDetail{
			TaskMeta: TaskMeta{
				Name:       t.Name,
				DataSource: src,
				Config:     t.Config,
			},
			State:       t.State,
			ProcessedBy: t.ProcessedBy,
		}
	}

	var wg sync.WaitGroup
	var lock sync.Mutex
	dErr := make(map[string]error)
	for _, d := range ds {
		wg.Add(1)
		go func(d *Detector) {
			ts, err := detectorAllTasks(d)
			lock.Lock()
			if err != nil {
				dErr[d.Host] = err
			} else {
				for _, t := range ts {
					if tasks[t.Name].ProcessedBy == d.Host {
						tasks[t.Name] = t
					}
				}
			}
			lock.Unlock()
			wg.Done()
		}(d)
	}
	wg.Wait()

	// merge results
	for _, t := range tasks {
		if err := dErr[t.ProcessedBy]; err != nil {
			t.Error = err.Error()
		}
	}

	summary := make(map[string]int)
	for _, t := range tasks {
		summary["task_"+t.State]++
		for _, ts := range t.Timeseries {
			summary["ts_"+ts.State]++
		}
	}

	c.JSON(200, map[string]interface{}{
		"summary": summary,
		"detail":  tasks,
	})
}

// QueryTaskDetail .
func QueryTaskDetail(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		c.String(400, "no task name")
		return
	}

	t, err := GetTaskByName(name)
	if err != nil {
		c.String(500, err.Error())
		return
	}

	// return directly if this task is stopped
	if t.State == TaskStopped {
		c.JSON(200, map[string]interface{}{
			"name":        t.Name,
			"data_source": json.RawMessage(t.DataSource),
			"config":      t.Config,
			"state":       t.State,
		})
		return
	}

	detail, err := taskDetail(t)
	if err != nil {
		c.String(500, err.Error())
		return
	}

	c.JSON(200, detail)
}

func taskDetail(t *Task) (*TaskDetail, error) {
	url := fmt.Sprintf("http://%v:%v/tsad/api/detector/query_task_detail?name=%v", t.ProcessedBy, config.WorkerPort, t.Name)
	var detail TaskDetail
	if err := GetModel(url, &detail); err != nil {
		return nil, err
	}
	return &detail, nil
}

// ForecastTask .
func ForecastTask(c *gin.Context) {
	type req struct {
		Name  string    `json:"name"` // task name
		Begin time.Time `json:"begin"`
		End   time.Time `json:"end"`
	}
	var r req
	if err := c.BindJSON(&r); err != nil {
		c.String(400, "invalid request")
		return
	}

	if r.End.Sub(r.Begin) > time.Hour*24*15 {
		c.String(500, "to large interval to forecast")
		return
	}

	if r.Begin.After(r.End) {
		c.String(500, "begin is after end")
		return
	}

	t, err := GetTaskByName(r.Name)
	if err != nil {
		c.String(500, "query task err: %v", err)
		return
	}

	results, err := forecastTask(t, r.Begin, r.End)
	if err != nil {
		c.String(500, "forecast task %v err: %v", t.Name, err)
		return
	}

	c.JSON(200, results)
}

func forecastTask(t *Task, begin, end time.Time) ([]*ForecastTS, error) {
	req := map[string]interface{}{
		"name":  t.Name,
		"begin": begin,
		"end":   end,
	}
	url := fmt.Sprintf("http://%v:%v/tsad/api/detector/forecast_task", t.ProcessedBy, config.WorkerPort)
	var results []*ForecastTS
	if err := PostModel(url, req, &results); err != nil {
		return nil, err
	}

	return results, nil
}

// StopTask .
func StopTask(c *gin.Context) {
	type req struct {
		Name string `json:"name"`
	}
	var r req
	if err := c.BindJSON(&r); err != nil {
		c.String(400, "invalid request")
		return
	}

	task, err := GetTaskByName(r.Name)
	if err != nil {
		c.String(500, "query task err: %v", err)
		return
	}

	// cancel this task
	if err := cancelTask(task); err != nil {
		c.String(500, "cancel task err: %v", err)
		return
	}

	// and update its state
	if err := UpdateTaskState(r.Name, TaskStopped); err != nil {
		c.String(500, "update task state err: %v", err)
		return
	}

	c.String(200, "ok")
}

func cancelTask(t *Task) error {
	req := map[string]string{"name": t.Name}
	url := fmt.Sprintf("http://%v:%v/tsad/api/detector/cancel_task", t.ProcessedBy, config.WorkerPort)
	return PostModel(url, req, nil)
}

// StartTask .
func StartTask(c *gin.Context) {
	type req struct {
		Name string `json:"name"`
	}
	var r req
	if err := c.BindJSON(&r); err != nil {
		c.String(400, "invalid request")
		return
	}

	task, err := GetTaskByName(r.Name)
	if err != nil {
		c.String(500, "query task err: %v", err)
		return
	}

	if task.State != TaskStopped {
		c.String(400, "task is not in stopped state")
		return
	}

	if err := UpdateTaskState(r.Name, TaskRunning); err != nil {
		c.String(500, "update task state err: %v", err)
		return
	}

	c.String(200, "ok")
}

// Summary .
func Summary(c *gin.Context) {
	ds, err := GetAliveDetectors()
	if err != nil {
		c.String(500, "get alice detectors err: %v", err)
		return
	}

	resutls := make(map[string]map[string]interface{}, len(ds))
	for _, d := range ds {
		sm, err := summary(d)
		resutls[d.Host] = make(map[string]interface{})
		if err != nil {
			resutls[d.Host]["error"] = err.Error()
		} else {
			resutls[d.Host]["summary"] = sm
		}
	}

	c.JSON(200, resutls)
}

func summary(d *Detector) (map[string]int, error) {
	url := fmt.Sprintf("http://%v:%v/tsad/api/detector/summary", d.Host, config.WorkerPort)
	var summary map[string]int
	if err := GetModel(url, &summary); err != nil {
		return nil, err
	}
	return summary, nil
}

// RetrainTask .
func RetrainTask(c *gin.Context) {
	type Req struct {
		Name string `json:"name"`
	}
	var r Req
	if err := c.BindJSON(&r); err != nil {
		c.String(400, "invalid arguments")
		return
	}

	task, err := GetTaskByName(r.Name)
	if err != nil {
		c.String(500, "query task err: %v", err)
		return
	}

	if err := retrainTask(task); err != nil {
		c.String(500, "retrain task err: %v", err)
		return
	}

	c.String(200, "ok")
}

func retrainTask(t *Task) error {
	req := map[string]string{"name": t.Name}
	url := fmt.Sprintf("http://%v:%v/tsad/api/detector/retrain_task", t.ProcessedBy, config.WorkerPort)
	return PostModel(url, req, nil)
}
