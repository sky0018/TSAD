package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"time"

	"code.byted.org/gopkg/stats"
	"code.byted.org/microservice/tsad/manager"
	"code.byted.org/microservice/tsad/worker"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	c, err := loadConfig()
	if err != nil {
		panic(fmt.Errorf("load config err: %v", err))
	}

	if err := stats.DoReport("toutiao.microservice.tsad"); err != nil {
		fmt.Printf("DoReport error: %s\n", err)
		panic(err)
	}
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%v", c.DebugPort), nil); err != nil {
			panic(err)
		}
	}()
	go func() {
		for range time.Tick(time.Minute * 5) {
			runtime.GC()
		}
	}()

	if err := manager.Start(&c.Manager); err != nil {
		panic(err)
	}

	if err := worker.Start(&c.Worker); err != nil {
		panic(err)
	}

	done := make(chan struct{})
	waitSignal(done)
	<-done
}

func waitSignal(done chan struct{}) {
	// TODO(zhangyuanjia)
}
