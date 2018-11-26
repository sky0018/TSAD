package tsmodels

import (
	"time"

	"code.byted.org/microservice/tsad/worker/ts"
)

/*
LineModel 直接将数据当做一条直线, 利用PerTrichotomyFit去拟合;
*/
type LineModel struct {
	src ts.TS
	// y = ax + b
	a  float64
	b  float64
	sd float64 // Standard Deviation
}

// NewLineModel .
func NewLineModel() *LineModel {
	return &LineModel{}
}

// Name .
func (lm *LineModel) Name() string {
	return "LineModel"
}

// SourceData .
func (lm *LineModel) SourceData() ts.TS {
	return lm.src
}

// Train .
func (lm *LineModel) Train(data ts.TS) error {
	lm.src = data
	XYPoints := ts.Points2XYPoints(lm.src.Points(), data.Begin(), time.Second)
	lm.a, lm.b, lm.sd = ts.PerTrichotomyFit(XYPoints, &ts.PerTrichotomyFitOp{
		MaxBadPer:        0.1,
		AngleIntervalNum: 10,
		AErr:             0.01,
		BErr:             0.01,
	})
	return nil
}

// Forecast .
func (lm *LineModel) Forecast(stamp time.Time) float64 {
	x := ts.Timestamp2X(stamp, lm.src.Begin(), time.Second)
	y := lm.a*x + lm.b
	return y
}

// ForecastInterval .
func (lm *LineModel) ForecastInterval(stamp time.Time) (float64, float64) {
	forecastY := lm.Forecast(stamp)
	return forecastY - (3 * lm.sd), forecastY + (3 * lm.sd)
}

// FeedLatest .
func (lm *LineModel) FeedLatest(points ts.Points) {
	// TODO(zhangyuanjia)
}
