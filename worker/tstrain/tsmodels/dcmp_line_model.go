package tsmodels

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"code.byted.org/microservice/tsad/worker/detector"
	"code.byted.org/microservice/tsad/worker/ts"
)

/*
DcmpLineModel .
	分解TS, 得到t, s, r;
	// 对t周期同位置点做拟合, 得到直线方程F;
	对t周期同位置点做tavg
	对s周期同位置点做savg;
	求r的ravg和rsd;

	则对点x的预测为, 其同周期的:
		expt: F(x) + savg(x) + ravg
		upper: expt + 3*rsd
		lower: expt - 3*rsd
*/
type DcmpLineModel struct {
	src    ts.TS
	trend  ts.TS
	season ts.TS
	random ts.TS

	Freq   int
	Begin  time.Time
	Period time.Duration

	// periodTrendLineA map[time.Duration]float64
	// periodTrendLineB map[time.Duration]float64
	PeriodTrend  map[time.Duration]float64
	PeriodSeaon  map[time.Duration]float64
	RandomAVG    float64
	RandomSD     float64
	LowerAdapter float64
	UpperAdapter float64

	BeyoundZero bool
}

// NewDcmpLineModel .
func NewDcmpLineModel() *DcmpLineModel {
	return &DcmpLineModel{}
}

// Name .
func (m *DcmpLineModel) Name() string {
	return "DcmpLineModel"
}

// ModelData .
func (m *DcmpLineModel) ModelData() ([]byte, error) {
	return json.Marshal(m)
}

// Recover .
func (m *DcmpLineModel) Recover(data []byte) error {
	var model DcmpLineModel
	if err := json.Unmarshal(data, &model); err != nil {
		return err
	}
	*m = model
	return nil
}

// IfBeyoundZero 如果时序的绝大多数值都大于等于0, 则预测值也需要大于等于0;
func (m *DcmpLineModel) IfBeyoundZero(data ts.TS) bool {
	beyound := 0
	for _, p := range data.Points() {
		if p.Value() >= 0 {
			beyound++
		}
	}
	return (float64(beyound) / float64(data.N())) >= 0.999
}

// Train .
func (m *DcmpLineModel) Train(data ts.TS, adapter detector.ModelAdapter) error {
	if data.Period() == ts.UnknownOrInvalid {
		return fmt.Errorf("no period")
	}
	if data.Frequency() == ts.UnknownOrInvalid {
		return fmt.Errorf("no frequency")
	}
	if !data.Completed() {
		// TODO(zhangyuanjia):
		//  complete this ts
		data = ts.LastValueComplete(data)
	}

	m.BeyoundZero = m.IfBeyoundZero(data)

	m.Freq = int(data.Period() / data.Frequency())
	m.src = data
	m.Begin = m.src.Begin()
	m.Period = m.src.Period()

	s, t, e, err := ts.ClassicalDecompose(m.src)
	if err != nil {
		return err
	}
	m.season = s
	m.trend = t
	m.random = e

	m.calPeriod()
	if err := m.adapt(adapter); err != nil {
		return fmt.Errorf("adapt err: %v", err)
	}

	m.clear()
	return nil
}

func (m *DcmpLineModel) clear() {
	// to save the memory
	m.src = nil
	m.season = nil
	m.trend = nil
	m.random = nil
}

func (m *DcmpLineModel) adapt(adapter detector.ModelAdapter) error {
	m.LowerAdapter = 3
	m.UpperAdapter = 3
	for i := 0; i < 30; i++ {
		if adapter(m.src, m.ForecastInterval) {
			break
		}
		m.LowerAdapter *= 2
		m.UpperAdapter *= 2
	}
	if !adapter(m.src, m.ForecastInterval) {
		return fmt.Errorf("can't find a max adapter")
	}

	var left, right float64 = 0, m.LowerAdapter
	cnt := 0
	for {
		m.LowerAdapter = (left + right) / 2
		m.UpperAdapter = (left + right) / 2
		if adapter(m.src, m.ForecastInterval) {
			if cnt > 40 || right-left < 0.0001 {
				break
			}
			right = m.UpperAdapter
		} else {
			left = m.UpperAdapter
		}
		cnt++
	}

	// adapt the lowerAdapter
	left, right = 0, m.LowerAdapter
	cnt = 0
	for {
		m.LowerAdapter = (left + right) / 2
		if adapter(m.src, m.ForecastInterval) {
			if cnt > 40 || right-left < 0.0001 {
				break
			}
			right = m.LowerAdapter
		} else {
			left = m.LowerAdapter
		}
		cnt++
	}

	// adapt the upperAdapter
	left, right = 0, m.UpperAdapter
	cnt = 0
	for {
		m.UpperAdapter = (left + right) / 2
		if adapter(m.src, m.ForecastInterval) {
			if cnt > 40 || right-left < 0.0001 {
				break
			}
			right = m.UpperAdapter
		} else {
			left = m.UpperAdapter
		}
		cnt++
	}

	return nil
}

func (m *DcmpLineModel) calPeriod() {
	pointMap, _ := ts.AggregatePeriodPoints(m.season)
	m.PeriodSeaon = make(map[time.Duration]float64, len(pointMap))
	for shift, ps := range pointMap {
		m.PeriodSeaon[shift] = ts.AVGPoints(ps)
	}

	pointMap, _ = ts.AggregatePeriodPoints(m.trend)
	m.PeriodTrend = make(map[time.Duration]float64, len(pointMap))
	for shift, ps := range pointMap {
		m.PeriodTrend[shift] = ts.AVGPoints(ps)
	}

	// m.periodTrendLineA = make(map[time.Duration]float64)
	// m.periodTrendLineB = make(map[time.Duration]float64)
	// for shift, ps := range pointMap {
	// 	a, b := ts.LSFit(ts.Points2XYPoints(ps, m.src.Begin(), m.src.Frequency()))
	// 	m.periodTrendLineA[shift] = a
	// 	m.periodTrendLineB[shift] = b
	// }

	avg := 0.0
	sum := 0.0
	for _, v := range m.random.Values() {
		avg += v
		sum += v * v
	}
	avg /= float64(m.random.N())
	sum /= float64(m.random.N())
	sd := math.Sqrt(sum)
	m.RandomAVG = avg
	m.RandomSD = sd
}

// Forecast .
func (m *DcmpLineModel) Forecast(timestamp time.Time) float64 {
	shift := ts.PeriodShift(timestamp, m.Begin, m.Period)
	// x := ts.Timestamp2X(m.src.Begin(), timestamp, m.src.Frequency())
	// a := m.periodTrendLineA[shift]
	// b := m.periodTrendLineB[shift]
	t := m.PeriodTrend[shift]
	s := m.PeriodSeaon[shift]
	result := t + s + m.RandomAVG
	if m.BeyoundZero && result < 0 {
		result = 0
	}
	return result
}

// ForecastInterval .
func (m *DcmpLineModel) ForecastInterval(timestamp time.Time) (lower, upper float64) {
	v := m.Forecast(timestamp)
	lower = v - m.LowerAdapter*m.RandomSD
	upper = v + m.UpperAdapter*m.RandomSD
	if m.BeyoundZero && lower < 0 {
		lower = 0
	}
	if m.BeyoundZero && upper < 0 {
		upper = 0
	}

	if lower == upper {
		upper += ((m.UpperAdapter + m.LowerAdapter) * m.RandomSD)
	}

	return
}
