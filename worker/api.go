package worker

import (
	"time"

	"code.byted.org/microservice/tsad/worker/detector"
	"github.com/gin-gonic/gin"
)

// ForecastTask .
func ForecastTask(c *gin.Context) {
	type req struct {
		Name  string    `json:"name"`
		Begin time.Time `json:"begin"`
		End   time.Time `json:"end"`
	}
	var r req
	if err := c.BindJSON(&r); err != nil {
		c.String(400, "invalid request")
		return
	}

	if r.End.Sub(r.Begin) > time.Hour*24*15 {
		c.String(400, "to large interval to forecast")
		return
	}

	// 30s per point
	r.Begin = leftShift30s(r.Begin)
	r.End = leftShift30s(r.End)
	stamps := make([]time.Time, 0, 100)
	for !r.Begin.After(r.End) {
		stamps = append(stamps, r.Begin)
		r.Begin = r.Begin.Add(time.Second * 30)
	}

	tss, err := detector.ForecastInterval(r.Name, stamps)
	if err != nil {
		c.String(500, "forecast interval err: %v", err)
		return
	}

	c.JSON(200, tss)
}

func leftShift30s(t time.Time) time.Time {
	unix := t.Unix()
	unix -= (unix % 30)
	return time.Unix(unix, 0)
}

// SubmitTask submit a task to the detector in this process
func SubmitTask(c *gin.Context) {
	var meta detector.TaskMeta
	if err := c.BindJSON(&meta); err != nil {
		c.String(400, "invalid argument")
		return
	}

	if !taskIsAllowed(meta) {
		c.String(200, "not in white datasource list")
		return
	}

	if err := detector.SubmitTask(meta); err != nil {
		c.String(500, "submit task err: %v", err)
		return
	}

	c.String(200, "submit task success")
}

// SubmitBatchTasks submit a batch of tasks to the detector in this process
func SubmitBatchTasks(c *gin.Context) {
	var batchTasks []detector.TaskMeta
	if err := c.BindJSON(&batchTasks); err != nil {
		c.String(400, "invalid argument")
		return
	}

	results := make([]string, 0, len(batchTasks))
	for _, t := range batchTasks {
		if taskIsAllowed(t) {
			err := detector.SubmitTask(t)
			if err == nil {
				results = append(results, "ok")
			} else {
				results = append(results, err.Error())
			}
		} else {
			results = append(results, "not in white datasource list")
		}
	}

	c.JSON(200, results)
}

// QueryTaskDetail .
func QueryTaskDetail(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		c.String(400, "no task name")
		return
	}
	t, ok := detector.TaskInfo(name)
	if !ok {
		c.String(500, "task %v not found", name)
		return
	}

	errMsg := ""
	var errStamp time.Time
	if err, stamp := t.Err(); err != nil {
		errMsg = err.Error()
		errStamp = stamp
	}
	c.JSON(200, map[string]interface{}{
		"name":             t.Name,
		"data_source":      t.DataSource,
		"state":            t.State(),
		"last_error":       errMsg,
		"last_error_stamp": errStamp,
	})
}

func AllTaskDetail(c *gin.Context) {
	tasks := detector.AllTasks()
	results := make([]map[string]interface{}, 0, len(tasks))
	for _, t := range tasks {
		errMsg := ""
		var errStamp time.Time
		if err, stamp := t.Err(); err != nil {
			errMsg = err.Error()
			errStamp = stamp
		}
		results = append(results, map[string]interface{}{
			"name":        t.Name,
			"data_source": t.DataSource,
			"state":       t.State(),
			"err":         errMsg,
			"err_stamp":   errStamp,
		})
	}

	c.JSON(200, results)
}

func AllTSDetail(c *gin.Context) {
	tasks := detector.AllTasks()
	results := make([]map[string]interface{}, 0, len(tasks))
	for _, t := range tasks {
		errMsg := ""
		var errStamp time.Time
		if err, stamp := t.Err(); err != nil {
			errMsg = err.Error()
			errStamp = stamp
		}

		ts := make(map[string]interface{})
		for _, s := range t.TSs() {
			errMsg := ""
			var errStamp time.Time
			if err, stamp := s.Err(); err != nil {
				errMsg = err.Error()
				errStamp = stamp
			}
			ts[s.Name()] = map[string]interface{}{
				"state":     s.State(),
				"err":       errMsg,
				"err_stamp": errStamp,
			}
		}

		results = append(results, map[string]interface{}{
			"name":        t.Name,
			"data_source": t.DataSource,
			"state":       t.State(),
			"err":         errMsg,
			"err_stamp":   errStamp,
			"ts":          ts,
		})
	}

	c.JSON(200, results)
}

// CancelTask .
func CancelTask(c *gin.Context) {
	type req struct {
		Name string `json:"name"`
	}
	var r req
	if err := c.BindJSON(&r); err != nil {
		c.String(400, "invalid request")
		return
	}

	if err := detector.CancelTask(r.Name); err != nil {
		c.String(500, err.Error())
		return
	}

	c.String(200, "ok")
}

// RetrainTask .
func RetrainTask(c *gin.Context) {
	type req struct {
		Name string `json:"name"`
	}
	var r req
	if err := c.BindJSON(&r); err != nil {
		c.String(400, "invalid request")
		return
	}

	t, ok := detector.TaskInfo(r.Name)
	if !ok {
		c.String(500, "no such a task %v", t.Name)
		return
	}

	key, err := srcKey(t.DataSource)
	if err != nil {
		c.String(500, "hash datasource err: %v", err)
		return
	}

	if err := DeleteModelData(key); err != nil {
		c.String(500, "delete old model data err: %v", err)
		return
	}

	if err := detector.CancelTask(t.Name); err != nil {
		c.String(500, "cancel task err: %v", err)
		return
	}

	c.String(200, "ok")
}

// Summary .
func Summary(c *gin.Context) {
	tasks := detector.AllTasks()
	summary := make(map[string]int)
	for _, t := range tasks {
		summary["task_"+string(t.State())]++
		for _, ts := range t.TSs() {
			summary["ts_"+string(ts.State())]++
		}
	}
	c.JSON(200, summary)
}
