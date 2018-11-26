package tsdb

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"code.byted.org/gopkg/taskqueue"
)

// Future .
type Future struct {
	QueryURL string
	Cost     time.Duration
	Err      error
	Model    interface{}
	Done     chan struct{}
}

func newFuture(qurl string, result *taskqueue.Result) *Future {
	done := make(chan struct{}, 1)
	future := &Future{
		Done:     done,
		QueryURL: qurl,
	}
	go func() {
		<-result.Done
		future.Cost = result.TimeCost
		future.Err = result.Error
		future.Model = result.Resp
		future.Done <- struct{}{}
	}()
	return future
}

func newErrFuture(qurl string, err error) *Future {
	done := make(chan struct{}, 1)
	done <- struct{}{}
	return &Future{
		Err:      err,
		Done:     done,
		QueryURL: qurl,
	}
}

const (
	// DefaultMaxConcurrency .
	DefaultMaxConcurrency = 1000
	defaultTaskBuffer     = 20000

	// DefaultTimeoutInMs .
	DefaultTimeoutInMs = 500
	// DefaultRetry .
	DefaultRetry = 0
)

// Options .
type Options struct {
	MaxConcurrency       int
	DefaultTimeoutInMs   int
	DefaultRetry         int
	DefaultRetryInterval time.Duration
	Logger               Logger
}

// Client .
type Client struct {
	options  *Options
	taskQue  *taskqueue.TaskQueue
	httpClis map[int]*http.Client // [timeoutInMs]Client, protected by lock
	lock     sync.RWMutex
}

// NewClient .
func NewClient(opt *Options) (*Client, error) {
	if opt == nil {
		return nil, fmt.Errorf("No options")
	}

	if opt.MaxConcurrency <= 0 || opt.MaxConcurrency > DefaultMaxConcurrency {
		return nil, fmt.Errorf("invalid max concurrency which should be in [0, %v]", DefaultMaxConcurrency)
	}
	if opt.DefaultTimeoutInMs == 0 {
		return nil, fmt.Errorf("invalid timeout: %v", opt.DefaultTimeoutInMs)
	}
	if opt.DefaultRetry < 0 {
		return nil, fmt.Errorf("invalid retry: %v", opt.DefaultRetry)
	}
	if opt.Logger == nil {
		opt.Logger = DefaultLogger
	}

	taskQue := taskqueue.NewTaskQueue(opt.MaxConcurrency, defaultTaskBuffer)
	taskQue.Start()
	return &Client{
		options:  opt,
		taskQue:  taskQue,
		httpClis: make(map[int]*http.Client),
	}, nil
}

// Do 同步的拉取数据
func (c *Client) Do(ctx context.Context, url string, model interface{}) error {
	future := c.Go(ctx, url, model)
	<-future.Done
	return future.Err
}

// Go 异步的提交一个请求
func (c *Client) Go(ctx context.Context, url string, model interface{}) *Future {
	type args struct {
		ctx   context.Context
		url   string
		model interface{}
	}
	result, err := c.taskQue.Submit(func(req interface{}) (interface{}, error) {
		a := req.(*args)
		err := c.query(a.ctx, a.url, a.model)
		return a.model, err
	}, &args{ctx, url, model})

	if err != nil {
		return newErrFuture(url, err)
	}

	return newFuture(url, result)
}

func (c *Client) query(ctx context.Context, url string, model interface{}) error {
	client := c.getClient(ctx)

	retry, ok := GetRetry(ctx)
	if !ok {
		retry = c.options.DefaultRetry
	}
	retryInterval, ok := GetRetryInterval(ctx)
	if !ok {
		retryInterval = c.options.DefaultRetryInterval
	}

	metricsStr, metricsOK := GetMetrics(ctx)

	var resp *http.Response
	var err error
	for i := 0; i < retry+1; i++ {
		if i > 0 {
			time.Sleep(retryInterval)
		}

		begin := time.Now()
		resp, err = client.Get(url)
		cost := time.Now().Sub(begin)
		costInUs := int(cost / 1000)

		// do metrics for this query
		if metricsOK {
			var throughput, latency string
			if err == nil {
				throughput = metricsStr + ".success.throughput"
				latency = metricsStr + ".success.latency"
			} else {
				throughput = metricsStr + ".error.throughput"
				latency = metricsStr + ".error.latency"
			}

			tags, _ := GetMetricsTags(ctx)
			metricsCli.EmitCounter(throughput, 1, "", tags)
			metricsCli.EmitTimer(latency, costInUs, "", tags)
		}

		if err != nil {
			continue
		}

		buf, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			continue
		}

		if err := json.Unmarshal(buf, model); err != nil {
			continue
		}

		return nil
	}

	return err
}

func (c *Client) getClient(ctx context.Context) *http.Client {
	timeout, ok := GetTimeout(ctx)
	if !ok {
		timeout = time.Duration(c.options.DefaultTimeoutInMs) * time.Millisecond
	}
	timeoutInMs := int(timeout / time.Millisecond)

	c.lock.RLock()
	httpCli, ok := c.httpClis[timeoutInMs]
	c.lock.RUnlock()
	if ok {
		return httpCli
	}

	httpCli = &http.Client{Timeout: time.Duration(timeoutInMs) * time.Millisecond}
	c.lock.Lock()
	if cli, ok := c.httpClis[timeoutInMs]; ok {
		c.lock.Unlock()
		return cli
	}
	c.httpClis[timeoutInMs] = httpCli
	c.lock.Unlock()

	return httpCli
}

// Close .
func (c *Client) Close() {
	c.taskQue.Stop()
}
