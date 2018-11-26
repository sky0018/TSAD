package ts

import (
	"fmt"
	"time"
)

// PeriodShift 得到stamp在其所在周期内的位移
func PeriodShift(stamp, begin time.Time, period time.Duration) time.Duration {
	diff := stamp.Sub(begin) % period
	if diff < 0 {
		diff += period
	}
	return diff
}

// AggregatePeriodPoints .
//  聚合TS中, 所有周期同位置的点;
//  返回值为 [shift]points, shift为各点相对于所在周期的起始位移
func AggregatePeriodPoints(data TS) (map[time.Duration]Points, error) {
	if data.Period() == UnknownOrInvalid {
		return nil, fmt.Errorf("period UnknownOrInvalid")
	}

	periodPs := make(map[time.Duration]Points)

	for _, p := range data.Points() {
		shift := PeriodShift(p.Stamp(), data.Begin(), data.Period())
		periodPs[shift] = append(periodPs[shift], p)
	}

	return periodPs, nil
}
