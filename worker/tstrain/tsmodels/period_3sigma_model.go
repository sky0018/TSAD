package tsmodels

import (
	"fmt"
	"time"

	"code.byted.org/microservice/tsad/worker/ts"
)

/*
Period3SigmaModel .
	选取各个中期中, 同位置的点, 形成一系列的子序列, subTS;
	对于每个subTS, 分别做线性拟合, 得到一次方程F;
	利用MinKSDValues和F, 得到subTS中拟合最好的一些点, 设这些点为bestSubTS;

	则, 对于任意点p, 设其同位置bestSubTS得到的均值和方差为avg, sd;
	expect = avg
	upper = avg + 3*sd
	lower = avg - 3*sd
*/
type Period3SigmaModel struct {
	src       ts.TS
	periodAs  map[time.Duration]float64 // 所有周期同位置拟合后的斜率
	periodBs  map[time.Duration]float64 // 所有周期同位置拟合后的常数
	periodSDs map[time.Duration]float64 // 所有周期同位置点标准差
	positions []time.Duration
}

// NewPeriod3SigmaModel .
func NewPeriod3SigmaModel() *Period3SigmaModel {
	return &Period3SigmaModel{}
}

// Name .
func (snm *Period3SigmaModel) Name() string {
	return "Period3SigmaModel"
}

// SourceData .
func (snm *Period3SigmaModel) SourceData() ts.TS {
	return snm.src
}

// Train .
func (snm *Period3SigmaModel) Train(data ts.TS) error {
	periodPoints, err := ts.AggregatePeriodPoints(data)
	if err != nil {
		return fmt.Errorf("AggregatePeriodPoints err: %v", err)
	}

	snm.src = data
	snm.periodAs = make(map[time.Duration]float64)
	snm.periodBs = make(map[time.Duration]float64)
	snm.periodSDs = make(map[time.Duration]float64)
	snm.positions = make([]time.Duration, 0, len(periodPoints))
	for shift, points := range periodPoints {
		xyPoints := ts.Points2XYPoints(points, data.Begin(), time.Second)
		a, b, sd := ts.PerTrichotomyFit(xyPoints, &ts.PerTrichotomyFitOp{
			MaxBadPer:        0.1,
			AngleIntervalNum: 5,
			AErr:             0.01,
			BErr:             0.01,
		}) // y = ax+b

		snm.periodAs[shift] = a
		snm.periodBs[shift] = b
		snm.periodSDs[shift] = sd
		snm.positions = append(snm.positions, shift)
	}

	return nil
}

// Forecast .
func (snm *Period3SigmaModel) Forecast(stamp time.Time) float64 {
	pos := ts.PeriodShift(stamp, snm.src.Begin(), snm.src.Period())
	if _, ok := snm.periodAs[pos]; !ok {
		left := snm.binaryFind(pos)
		pos = snm.positions[left]
	}

	a := snm.periodAs[pos]
	b := snm.periodBs[pos]
	x := ts.Timestamp2X(stamp, snm.src.Begin(), time.Second)
	return a*x + b
}

// ForecastInterval .
func (snm *Period3SigmaModel) ForecastInterval(stamp time.Time) (float64, float64) {
	pos := ts.PeriodShift(stamp, snm.src.Begin(), snm.src.Period())
	if _, ok := snm.periodAs[pos]; !ok {
		left := snm.binaryFind(pos)
		pos = snm.positions[left]
	}

	a := snm.periodAs[pos]
	b := snm.periodBs[pos]
	x := ts.Timestamp2X(stamp, snm.src.Begin(), time.Second)
	expt := a*x + b
	sd := snm.periodSDs[pos]
	return expt - 3*sd, expt + 3*sd
}

func (snm *Period3SigmaModel) binaryFind(pos time.Duration) (left int) {
	left = 0
	right := len(snm.positions)
	for left < right {
		mid := (left + right) >> 1
		if snm.positions[mid] > pos {
			right = mid
		} else {
			left = mid
		}
	}
	return left
}

// FeedLatest .
func (snm *Period3SigmaModel) FeedLatest(points ts.Points) {
	// TODO(zhangyuanjia)
}
