package ts

import (
	"fmt"
	"time"
)

/*
MovingLinearClean .
	移动线性清洗

	扫描每个点P(该点或缺失), 选取以该点为中心, 一定窗口范围内的所有点, 形成点集Ps;
	对Ps做线性拟合, 得到方程F;
	利用F和MinKSDValues, 选取Ps中拟合最好的那一部分点集BestPs;
	计算BestPs的avg和sd;

	如果点P的值在 [avg-3*sd, avg+3*sd] 或者 P点值缺失, 则将P点值设置为 avg;

	// TODO(zhangyuanjia): 根据数据自动选取训练的误差参数
*/
func MovingLinearClean(data TS, halfWindow time.Duration) (TS, error) {
	if data.N() == 0 {
		return nil, fmt.Errorf("empty time-series")
	}

	points := data.Points()
	var left, right int
	cleanedPoints := make(Points, 0, len(points))
	for stamp := data.Begin(); !stamp.After(data.End()); stamp = stamp.Add(data.Frequency()) {
		for points[left].Stamp().Add(halfWindow).Before(stamp) {
			left++
		}
		for right < len(points) && stamp.Add(halfWindow).After(points[right].Stamp()) {
			right++
		}

		window := points[left:right]
		xyPoints := Points2XYPoints(window, window[0].Stamp(), time.Second)
		a, b, sd := PerTrichotomyFit(xyPoints, &PerTrichotomyFitOp{
			MaxBadPer:        0.1,
			AngleIntervalNum: 2,
			AErr:             0.01,
			BErr:             1,
		})

		x := Timestamp2X(stamp, window[0].Stamp(), time.Second)
		ecpt := a*x + b
		var val float64
		if p, ok := data.GetPoint(stamp); !ok {
			val = ecpt
		} else {
			if p.Value() < ecpt-3*sd || p.Value() > ecpt+3*sd {
				val = ecpt
			} else {
				val = p.Value()
			}
		}

		cleanedPoints = append(cleanedPoints, NewPoint(stamp, val))
	}

	return NewTS(data.Attributes(), cleanedPoints), nil
}
