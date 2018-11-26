package worker

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"code.byted.org/microservice/tsad/utils"
)

var (
	metricser utils.Metricser
	config    *Config
)

// Start .
func Start(c *Config) error {
	config = c

	metricser = utils.NewDefaultMetricser()

	if err := initDAL(); err != nil {
		return err
	}
	if err := initFetchers(); err != nil {
		return err
	}
	if err := startDetector(); err != nil {
		return err
	}
	if err := initWhiteBlackList(c); err != nil {
		return err
	}

	g := gin.Default()
	tsadAPI := g.Group("tsad/api")
	// API for detector
	det := tsadAPI.Group("detector")
	{
		det.POST("submit_task", SubmitTask)
		det.POST("submit_batch_tasks", SubmitBatchTasks)
		det.GET("query_task_detail", QueryTaskDetail)
		det.GET("all_task_detail", AllTaskDetail)
		det.GET("all_ts_detail", AllTSDetail)
		det.POST("cancel_task", CancelTask)
		det.POST("forecast_task", ForecastTask)
		det.POST("retrain_task", RetrainTask)
		det.GET("summary", Summary)
	}

	go func() {
		err := g.Run(fmt.Sprintf("0.0.0.0:%v", c.WorkerPort))
		panic(err)
	}()

	return nil
}
