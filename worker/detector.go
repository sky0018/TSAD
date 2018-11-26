package worker

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"code.byted.org/gopkg/tsdb"
	"code.byted.org/microservice/tsad/worker/detector"
	"code.byted.org/microservice/tsad/worker/ts"
	"code.byted.org/microservice/tsad/worker/tsfetcher"
	"code.byted.org/microservice/tsad/worker/tspreprocess"
	"code.byted.org/microservice/tsad/worker/tstrain"
	"code.byted.org/gopkg/env"
	"code.byted.org/microservice/tsad/utils"
)

var (
	tsdbClient *tsdb.Client
	logger     = utils.NewLogger("worker")
)

func startDetector() error {
	var err error
	tsdbClient, err = tsdb.NewClient(&tsdb.Options{
		MaxConcurrency:       1000,
		DefaultTimeoutInMs:   int((time.Second * 10) / time.Microsecond),
		DefaultRetry:         5,
		DefaultRetryInterval: time.Second,
	})
	if err != nil {
		return fmt.Errorf("create tsdb client err: %v", err)
	}

	taskLeaser, err := NewDefaultTaskLeaser(config.MysqlDSN)
	if err != nil {
		return fmt.Errorf("NewDefaultTaskLeaser err: %v", err)
	}
	op := &detector.Options{
		MaxTasks: config.MaxTasks,
		P: &detector.Plugins{
			Heartbeat:   Heartbeat,
			FetchFromTo: FetchFromTo,
			// fetchFromTo: FetchFrom
			DeriveSource:   DeriveSource,
			Train:          Train,
			StoreModelData: StoreModelData,
			ReadModelData:  ReadModelData,
			RecoverModel:   RecoverModel,
			Preprocess:     Preprocess,
			ModelAdapter:   ModelAdapter,
			Alert:          Alert,
		},
		TaskLeaser: taskLeaser,
	}

	return detector.Start(op)
}

var (
	tsdbFetcher tsfetcher.TSFetcher
)

func initFetchers() error {
	var err error
	middleStore, err := tsfetcher.NewMysqlMiddleStore(config.MysqlDSN)
	if err != nil {
		return fmt.Errorf("NewMysqlMiddleStore err: %v", err)
	}

	tsdbFetcher, err = tsfetcher.NewTSDBFetcher(config.TSDBAPI,
		config.TSDBRetry,
		config.TSDBTimeout*time.Millisecond, middleStore)
	if err != nil {
		return fmt.Errorf("NewTSDBFetcher err: %v", err)
	}
	return nil
}

// FetchFromTo .
func FetchFromTo(ctx context.Context, tms *detector.TimeSeries, from, to time.Time) (ts.TS, error) {
	end := time.Now().Add(-time.Minute * 1) // the latest data is inaccurate
	if to.After(end) {
		to = end
	}
	if to.Before(from) {
		return nil, fmt.Errorf("latest data is inaccurate, please try later")
	}

	fchSrc, err := dataSource2FetcherSource(tms.DataSource)
	if err != nil {
		return nil, fmt.Errorf("invalid source and ts: %v, err: %v", tms, err)
	}
	switch tms.DataSource.Type {
	case detector.DataSourceTypeTSDB:
		return tsdbFetcher.Fetch(ctx, fchSrc, from, to)
	}

	return nil, fmt.Errorf("invalid DataSourceType: %v", tms.DataSource.Type)
}

func dataSource2FetcherSource(source detector.DataSource) (tsfetcher.Source, error) {
	switch source.Type {
	case detector.DataSourceTypeTSDB:
		return tsfetcher.Source{
			Type:  tsfetcher.SourceTSDB,
			Key:   source.Key,
			Extra: source.Extra,
		}, nil
	}
	return tsfetcher.Source{}, fmt.Errorf("unknown data source type: %v", source.Type)
}

// DeriveSource .
func DeriveSource(src detector.DataSource) ([]detector.DataSource, error) {
	if src.Type != detector.DataSourceTypeTSDB {
		return []detector.DataSource{src}, nil
	}

	if src.Type == detector.DataSourceTypeTSDB {
		if !strings.Contains(src.Extra, "*") {
			return []detector.DataSource{src}, nil
		}
	}

	pattern := "%v?start=10m-ago&m=%v%v"
	tsdbURL := fmt.Sprintf(pattern, config.TSDBAPI, src.Key, src.Extra)
	var resp []*tsdb.RespModel
	if err := tsdbClient.Do(context.Background(), tsdbURL, &resp); err != nil {
		return nil, fmt.Errorf("query tsdb err: %v", err)
	}

	var srcs []detector.DataSource
	for _, r := range resp {
		var kvs []string
		for k, v := range r.Tags {
			kvs = append(kvs, fmt.Sprintf("%v=%v", k, v))
		}
		extra := "{" + strings.Join(kvs, ",") + "}"
		srcs = append(srcs, detector.DataSource{
			Type:  src.Type,
			Key:   src.Key,
			Extra: extra,
		})
	}

	return srcs, nil
}

// Heartbeat .
func Heartbeat(numTasks int) error {
	return UpdateDetectorInfo(&Detector{
		Host:      env.HostIP(),
		NumTasks:  numTasks,
		HeartBeat: time.Now(),
	})
}

// Train .
func Train(data ts.TS, adapter detector.ModelAdapter) (detector.TSModel, error) {
	return tstrain.Train(data, adapter)
}

func srcKey(src detector.DataSource) (string, error) {
	// src string may be too long, so do md5 for it
	srcStr := src.String()
	h := md5.New()
	_, err := h.Write([]byte(srcStr))
	if err != nil {
		return "", err
	}
	srcMd5 := h.Sum(nil)
	str := base64.StdEncoding.EncodeToString(srcMd5)
	return str, nil
}

// StoreModelData .
func StoreModelData(src detector.DataSource, mname, data string,
	trainStamp time.Time) error {
	key, err := srcKey(src)
	if err != nil {
		return err
	}

	m := &ModelData{
		SrcKey: key,
		Name:   mname,
		Data:   data,
		Stamp:  time.Now(),
	}
	return SaveModelData(m)
}

// ReadModelData .
func ReadModelData(src detector.DataSource) (mname string, data string, trainStamp time.Time, err error) {
	key, err := srcKey(src)
	if err != nil {
		return "", "", time.Now(), err
	}

	m, err := QueryModelData(key)
	if err != nil {
		return "", "", time.Now(), err
	}
	return m.Name, m.Data, m.Stamp, nil
}

// RecoverModel .
func RecoverModel(name string, data []byte) (detector.TSModel, error) {
	return tstrain.Recover(name, data)
}

// Preprocess .
func Preprocess(data ts.TS) (ts.TS, error) {
	return tspreprocess.Preprocess(data)
}

// ModelAdapter .
func ModelAdapter(data ts.TS,
	forecast func(timestamp time.Time) (lower, upper float64)) (accepted bool) {
	n := data.N()
	errCnt := 0
	for _, p := range data.Points() {
		l, u := forecast(p.Stamp())
		if p.Value() < l || p.Value() > u {
			errCnt++
		}
	}

	errPer := float64(errCnt) / float64(n)
	if errPer > (1 - 0.999) {
		return false
	}

	return true
}

// Alarm .
type Alarm struct {
	Name       string    `json:"name"`
	DataSource string    `json:"data_source"`
	Stamp      time.Time `json:"stamp"`
	Observed   float64   `json:"observed"`
	Lower      float64   `json:"lower"`
	Upper      float64   `json:"upper"`
}

// MSAlert .
type MSAlert struct {
	RuleID uint   `json:"rule_id"`
	Sender string `json:"sender"`

	Content    string            `json:"content"`
	Vars       map[string]string `json:"vars"`
	Tags       map[string]string `json:"tags"`
	Metrics    []string          `json:"metrics"`
	DetailURL  string            `json:"detail_url"`
	SenderHost string            `json:"sender_host"`
}

// Alert .
func Alert(t *detector.Task, ts *detector.TimeSeries,
	lower, upper float64, ob ts.Point) {

	src, _ := json.Marshal(ts.DataSource)
	alarm := &Alarm{
		Name:       ts.TaskName,
		DataSource: string(src),
		Stamp:      ob.Stamp(),
		Observed:   ob.Value(),
		Lower:      lower,
		Upper:      upper,
	}
	buf, _ := json.Marshal(alarm)
	logger.Infof(">>>>> alert", string(buf))

	// TODO(zhangyuanjia): send alert to msbackend
	rid, err := strconv.Atoi(t.Name)
	if err != nil {
		logger.Errorf("invalid ms ruleid ", t.Name)
		return
	}
	ma := &MSAlert{
		RuleID: uint(rid),
		Sender: "tsad",
		Vars: map[string]string{
			"upper":   fmt.Sprintf("%v", upper),
			"lower":   fmt.Sprintf("%v", lower),
			"observe": fmt.Sprintf("%v", ob.Value()),
		},
		Content: fmt.Sprintf("tsad alert: %v", ts.DataSource),
	}
	buf, _ = json.Marshal(ma)
	resp, err := http.Post("http://ms.byted.org/msbackend/api/alarm/gen_alarm", "application/json", bytes.NewReader(buf))
	if err != nil {
		logger.Errorf("post alert err: %v", err)
		return
	}

	buf, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf("read ms alert resp err: %v", err)
		return
	}
	logger.Infof("post ms alert: ", string(buf))

	return
}

// DefaultTaskLeaser .
type DefaultTaskLeaser struct {
	distLocker DistLocker
}

// NewDefaultTaskLeaser .
func NewDefaultTaskLeaser(dsn string) (*DefaultTaskLeaser, error) {
	op := &MysqlDistLockOptions{
		Identity:        env.HostIP(),
		DSN:             dsn,
		TableName:       "tsad_tasks",
		KeyField:        "name",
		LockedByField:   "processed_by",
		ExpirationField: "lock_expiration",
	}
	distLocker, err := NewMysqlDistLocker(op)
	if err != nil {
		return nil, fmt.Errorf("newDistLocker err: %v", err)
	}
	return &DefaultTaskLeaser{
		distLocker: distLocker,
	}, nil
}

// Lease .
// default Lease func use the mysql update
func (leaser DefaultTaskLeaser) Lease(taskName string, lease time.Duration) error {
	return leaser.distLocker.LockLease(taskName, lease)
}

// Renewal .
func (leaser DefaultTaskLeaser) Renewal(taskName string, lease time.Duration) error {
	return leaser.distLocker.RenewalLease(taskName, lease)
}

// Unlease .
func (leaser DefaultTaskLeaser) Unlease(taskName string) error {
	return leaser.distLocker.Unlock(taskName)
}
