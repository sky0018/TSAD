package manager

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

var (
	config *Config
)

// OPTIONSHandle .
func OPTIONSHandle(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Methods", c.Request.Header.Get("Access-Control-Request-Method"))
	c.Header("Access-Control-Allow-Credentials", "true")
	c.Header("Access-Control-Allow-Headers", c.Request.Header.Get("Access-Control-Request-Headers"))
}

// AllowControl .
func AllowControl() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Next()
	}
}

// Start .
func Start(c *Config) error {
	config = c
	if err := initDAL(); err != nil {
		return err
	}
	if err := initDistLock(); err != nil {
		return err
	}

	go startTaskDister()

	g := gin.Default()
	g.Use(AllowControl())
	g.OPTIONS("tsad/api/*pattern", OPTIONSHandle)
	tsadAPI := g.Group("tsad/api")

	tsadAPI.POST("submit_task", SubmitTask)
	tsadAPI.POST("forecast_task", ForecastTask)
	tsadAPI.POST("update_task", UpdateTask)
	tsadAPI.GET("query_task_detail", QueryTaskDetail)
	tsadAPI.GET("all_task_detail", AllTaskDetail)
	tsadAPI.POST("stop_task", StopTask)
	tsadAPI.POST("start_task", StartTask)
	tsadAPI.POST("retrain_task", RetrainTask)
	tsadAPI.GET("summary", Summary)

	go func() {
		err := g.Run(fmt.Sprintf("0.0.0.0:%v", config.ManagerPort))
		panic(err)
	}()
	return nil
}
