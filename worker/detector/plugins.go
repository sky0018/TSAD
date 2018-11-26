package detector

import (
	"context"
	"fmt"
	"time"

	"code.byted.org/microservice/tsad/worker/ts"
)

// ModelAdapter .
type ModelAdapter func(src ts.TS,
	forecast func(timestamp time.Time) (lower, upper float64)) (accepted bool)

// TSModel used to forecast time-series
type TSModel interface {
	Name() string
	Train(data ts.TS, adapter ModelAdapter) error
	Forecast(timestamp time.Time) float64
	ForecastInterval(timestamp time.Time) (lower, upper float64)

	ModelData() ([]byte, error)
	Recover(data []byte) error
	// SourceData() ts.TS
	// FeedLatest(points ts.Points)        // append the latest points to this model
	// Retrain(adapter ModelAdapter) error // retrain this model after feed some latest points
}

// Plugins .
type Plugins struct {
	/*
		task中输入的src, 或许会对于多条ts;
		如: metric{host=*};
		DeriveSource将该src展开成多个srcs, 展开后的src对应单条ts;
		如:
			[metric{hots=127.0.0.1}, metrics{host=127.0.0.2}]
	*/
	DeriveSource func(src DataSource) ([]DataSource, error)

	// Heartbeat .
	Heartbeat func(numTasks int) error

	// funcs for fetching time-series data
	FetchFromTo func(ctx context.Context, ts *TimeSeries, from, to time.Time) (ts.TS, error)

	// train model from this data
	Train          func(data ts.TS, adapter ModelAdapter) (TSModel, error)
	StoreModelData func(src DataSource, mname, data string, trainStamp time.Time) error
	ReadModelData  func(src DataSource) (mname, data string, trainStamp time.Time, err error)
	RecoverModel   func(mname string, data []byte) (TSModel, error)

	// clean this time-series
	Preprocess func(data ts.TS) (ts.TS, error)

	// ModelAdapter .
	ModelAdapter ModelAdapter

	// alert anomaly for this point in the time-series
	Alert func(t *Task, ts *TimeSeries, lower, upper float64, ob ts.Point)
}

// Valid .
func (e Plugins) Valid() error {
	if e.DeriveSource == nil {
		return fmt.Errorf("no DeriveSource")
	}
	if e.Heartbeat == nil {
		return fmt.Errorf("no Heartbeat")
	}
	if e.FetchFromTo == nil {
		return fmt.Errorf("no FetchFromTo")
	}
	if e.Train == nil {
		return fmt.Errorf("no Train")
	}
	if e.Preprocess == nil {
		return fmt.Errorf("no Preprocess")
	}
	if e.ModelAdapter == nil {
		return fmt.Errorf("no ModelAdapter")
	}
	if e.Alert == nil {
		return fmt.Errorf("no Alert")
	}
	if e.StoreModelData == nil {
		return fmt.Errorf("no StoreModelData")
	}
	if e.ReadModelData == nil {
		return fmt.Errorf("no ReadModelData")
	}
	if e.RecoverModel == nil {
		return fmt.Errorf("no RecoverModel")
	}

	return nil
}

// TaskLeaser used to lock tasks
type TaskLeaser interface {
	Lease(taskName string, lease time.Duration) error
	Renewal(taskName string, lease time.Duration) error
	Unlease(taskName string) error
}
