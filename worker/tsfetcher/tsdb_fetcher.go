package tsfetcher

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"code.byted.org/gopkg/metrics"
	"code.byted.org/gopkg/tsdb"
	"code.byted.org/microservice/tsad/worker/ts"
	"code.byted.org/gopkg/logs"
)

// Tags .
type Tags map[string]string

// AggregateTags .
type AggregateTags []string

// DPS .
type DPS map[string]float64

type tsdbResp struct {
	Metric        string        `json:"metric"`
	Tags          Tags          `json:"tags"`
	AggregateTags AggregateTags `json:"aggregateTags"`
	DPS           DPS           `json:"dps"`
}

// TSDBTSAttr attr for tsdb time-series
var TSDBTSAttr = ts.Attributes{
	Frequency: time.Second * 30,
	Period:    time.Hour * 24, // fix period as 1 day
}

// TSDBFetcher .
type TSDBFetcher struct {
	client *tsdb.Client

	tsdbAPI string
	timeout time.Duration
	retry   int
	limiter *ConcLimiter
	metric  *metrics.MetricsClientV2

	middleStorer MiddleStorer
}

// NewTSDBFetcher .
func NewTSDBFetcher(tsdbAPI string, defaultRetry int, defaultTimeout time.Duration, ms MiddleStorer) (*TSDBFetcher, error) {
	op := &tsdb.Options{
		MaxConcurrency:     1000,
		DefaultTimeoutInMs: int(defaultTimeout / time.Millisecond),
		DefaultRetry:       defaultRetry,
	}
	client, err := tsdb.NewClient(op)
	if err != nil {
		return nil, fmt.Errorf("error when create tsdb.Client: %v", err)
	}

	limiter := NewConcLimiter(5)
	metric := metrics.NewDefaultMetricsClientV2("toutiao.microservice.tsad", true)
	return &TSDBFetcher{client, tsdbAPI, defaultTimeout, defaultRetry, limiter, metric, ms}, nil
}

// Fetch fetch data
// source's Extra is metrics tag, and the Extra con not contains *, so the tag must be definite
func (tf *TSDBFetcher) Fetch(ctx context.Context, src Source, begin, end time.Time) (TS ts.TS, rerr error) {
	var points ts.Points
	defer func() {
		if rerr == nil {
			points = complete(points, TSDBTSAttr.Frequency)
			TS = ts.NewTS(TSDBTSAttr, points)
			if tf.middleStorer != nil {
				if err := tf.middleStorer.Store(src, points); err != nil {
					logs.Errorf("[TSDBFethcer] middle storer err: %v", err)
				}
			}
		}
	}()

	// read from middle store
	if tf.middleStorer == nil {
		points, rerr = tf.fetchByDay(ctx, src, begin, end)
		return
	}

	oldPoints, err := tf.middleStorer.Fetch(src)
	if err != nil {
		logs.Errorf("[TSDBFethcer] read from middle storer err: %v", err)
		points, rerr = tf.fetchByDay(ctx, src, begin, end)
		return
	}

	for (len(oldPoints) > 0) && oldPoints[0].Stamp().Before(begin) {
		oldPoints = oldPoints[1:]
	}

	if len(oldPoints) == 0 {
		points, rerr = tf.fetchByDay(ctx, src, begin, end)
		return
	}

	newBegin := oldPoints[len(oldPoints)-1].Stamp().Add(TSDBTSAttr.Frequency)
	points, rerr = tf.fetchByDay(ctx, src, newBegin, end)
	if rerr != nil {
		return
	}

	points = append(oldPoints, points...)
	return
}

func (tf *TSDBFetcher) fetchByDay(ctx context.Context, source Source, begin, end time.Time) (ts.Points, error) {
	if source.Type != SourceTSDB {
		return nil, fmt.Errorf("not a tsdb source type: %v", SourceTSDB)
	}
	timeout := tf.timeout
	retry := tf.retry
	if t, ok := GetTimeout(ctx); ok {
		timeout = t
	}
	if r, ok := GetRetry(ctx); ok {
		retry = r
	}

	ctx = tsdb.WithTimeout(ctx, timeout)
	ctx = tsdb.WithRetry(ctx, retry)

	points := make(ts.Points, 0, 2880)
	dayDur := time.Hour * 24
	for begin.Before(end) {
		pos := begin.Add(dayDur)
		if pos.After(end) {
			pos = end
		}
		// dayPoints, err := tf.fetchConByHour(ctx, source, begin, pos)
		dayPoints, err := tf.fetch(ctx, source, begin, pos)
		if err != nil {
			return nil, fmt.Errorf("fetch start %v to %v error: %v", begin, pos, err)
		}

		points = append(points, dayPoints...)
		begin = pos
	}

	return points, nil
}

func (tf *TSDBFetcher) fetchConByHour(ctx context.Context, source Source, begin, end time.Time) (ts.Points, error) {
	batch := make([]ts.Points, 0, 24)
	batchErrs := make([]error, 0, 24)
	var wg sync.WaitGroup
	id := 0

	for begin.Before(end) {
		pos := begin.Add(time.Hour)
		if pos.After(end) {
			pos = end
		}

		batch = append(batch, nil)
		batchErrs = append(batchErrs, nil)
		wg.Add(1)
		go func(begin, end time.Time, id int) {
			points, err := tf.fetch(ctx, source, begin, end)
			batch[id] = points
			batchErrs[id] = err
			wg.Done()
		}(begin, pos, id)
		id++

		begin = pos
	}

	wg.Wait()
	for i := range batchErrs {
		if batchErrs[i] != nil {
			return nil, batchErrs[i]
		}
	}

	ps := make(ts.Points, 0, 2880)
	for i := range batch {
		ps = append(ps, batch[i]...)
	}
	return ps, nil
}

// fetch .
func (tf *TSDBFetcher) fetch(ctx context.Context, source Source, begin, end time.Time) (data ts.Points, rerr error) {
	tsdbURL, err := tf.source2tsdbURL(source, begin, end)
	if err != nil {
		return nil, fmt.Errorf("[source2tsdbURL]: %v", err)
	}

	defer func(begin time.Time) {
		cost := time.Now().Sub(begin)
		if rerr == nil {
			logs.Infof("[TSDBFetcher] query url=%v successfully, cost=%v", tsdbURL, cost)
		} else {
			logs.Errorf("[TSDBFetcher] query url=%v error, cost=%v, err=%v", tsdbURL, cost, rerr)
		}
	}(time.Now())

	v := ctx.Value("noblock")
	ctx = tsdb.WithMetrics(ctx, "toutiao.microservice.tsad")

	if v == nil {
		tf.limiter.Wait()
		defer tf.limiter.Release()
		tf.metric.EmitCounter("tsdb.query", 1)
	}

	var resp []*tsdbResp
	if err := tf.client.Do(ctx, tsdbURL, &resp); err != nil {
		return nil, fmt.Errorf("query tsdb err: %v", err)
	}

	if len(resp) == 0 {
		return nil, fmt.Errorf("no data in query: %v", tsdbURL)
	}

	return tsdbResp2Points(resp)
}

func (tf *TSDBFetcher) source2tsdbURL(source Source, begin, end time.Time) (string, error) {
	if source.Type != SourceTSDB {
		return "", fmt.Errorf("not a tsdb source type: %v", SourceTSDB)
	}
	if source.Key == "" {
		return "", fmt.Errorf("source Key can not be null")
	}
	if strings.Contains(source.Extra, "*") {
		return "", fmt.Errorf("source Extra can not contains *, Extra: %v", source.Extra)
	}
	startTime := tsdb.TimeStamp(begin)
	endTime := tsdb.TimeStamp(end)
	// url for example
	// http://my_host/api/query?start=2017-01-01 00:00:00&end=2017-01-01 01:00:00&m=sum:rate:metrics_key{host=10.1.10.1}
	url := fmt.Sprintf("%v?start=%v&end=%v&m=%v%v", tf.tsdbAPI, startTime, endTime, source.Key, source.Extra)
	return url, nil
}

func tsdbResp2Points(resp []*tsdbResp) ([]ts.Point, error) {
	if len(resp) != 1 {
		return nil, fmt.Errorf("tsdb response err, resp length must be 1, current resp length is : %v", len(resp))
	}
	// get all point from tsd resp
	points := make([]ts.Point, 0, 100)
	tsdbDPS := resp[0].DPS
	// sort the map with timestamp
	var keys []string
	for timestamp := range tsdbDPS {
		keys = append(keys, timestamp)
	}
	// sort the keys
	sort.Strings(keys)
	// make the time series sequentially
	for _, timestamp := range keys {
		i, err := strconv.ParseInt(timestamp, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse timestamp to int error, timestamp : %v, error: %v", timestamp, err)
		}
		tm := time.Unix(i, 0)
		points = append(points, ts.NewPoint(
			tm,
			tsdbDPS[timestamp],
		))
	}
	return points, nil
}
