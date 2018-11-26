package ts

import (
	"fmt"
	"time"
)

// ClassicalDecompose https://www.otexts.org/fpp/6/3
func ClassicalDecompose(data TS) (s, t, e TS, err error) {
	if data.Period() == UnknownOrInvalid {
		return nil, nil, nil, fmt.Errorf("no period")
	}
	if data.Frequency() == UnknownOrInvalid {
		return nil, nil, nil, fmt.Errorf("no frequency")
	}
	if !data.Completed() {
		// TODO(zhangyuanjia):
		//  complete this ts
		return nil, nil, nil, fmt.Errorf("not a completed ts")
	}

	// cal trend
	m := int(data.Period() / data.Frequency())
	ps := data.Points()

	var trendPs Points
	if m%2 > 0 { // odd number
		trendPs = MovingAVG(ps, m)
	} else { // even number
		trendPs = MovingAVG(ps, m)
		trendPs = MovingAVG(trendPs, m)
	}

	t = NewTS(data.Attributes(), trendPs)

	detrendedPs := make(Points, 0, len(trendPs))
	for _, tp := range trendPs {
		stamp := tp.Stamp()
		srcP, _ := data.GetPoint(stamp)
		detrendedPs = append(detrendedPs, NewPoint(stamp, srcP.Value()-tp.Value()))
	}

	// cal season
	detrendedTs := NewTS(data.Attributes(), detrendedPs)
	periodPoints, _ := AggregatePeriodPoints(detrendedTs)
	periodAVG := make(map[time.Duration]float64)
	for shift, ps := range periodPoints {
		periodAVG[shift] = AVGPoints(ps)
	}

	seasonPs := make(Points, 0, len(trendPs))
	for _, p := range trendPs {
		shift := PeriodShift(p.Stamp(), data.Begin(), data.Period())
		seasonPs = append(seasonPs, NewPoint(p.Stamp(), periodAVG[shift]))
	}

	s = NewTS(data.Attributes(), seasonPs)

	// cal random
	randomPs := make(Points, 0, len(trendPs))
	for i := range trendPs {
		stamp := trendPs[i].Stamp()
		srcP, _ := data.GetPoint(stamp)
		randomPs = append(randomPs, NewPoint(stamp, srcP.Value()-trendPs[i].Value()-seasonPs[i].Value()))
	}

	e = NewTS(data.Attributes(), randomPs)
	return
}

// MovingAVG .
func MovingAVG(ps Points, window int) Points {
	sum := 0.0
	for i := 0; i < window; i++ {
		sum += ps[i].Value()
	}

	results := make(Points, 0, len(ps)-window+1)
	results = append(results, NewPoint(
		ps[window-1-window/2].Stamp(),
		sum/float64(window),
	))

	for i := window; i < len(ps); i++ {
		sum += ps[i].Value()
		sum -= ps[i-window].Value()
		results = append(results, NewPoint(
			ps[i-window/2].Stamp(),
			sum/float64(window),
		))
	}

	return results
}
