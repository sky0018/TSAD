package ts

import (
	"math"
	"sort"
	"time"
)

// XYPoint .
type XYPoint struct {
	X float64
	Y float64
}

// XYPoints .
type XYPoints []XYPoint

// SortByX .
func (ps XYPoints) SortByX() {
	sort.Slice(ps, func(i, j int) bool {
		return ps[i].X < ps[j].X
	})
}

// Timestamp2X .
func Timestamp2X(begin, stamp time.Time, unit time.Duration) float64 {
	return float64(stamp.Sub(begin) / unit)
}

// Point2XYPoint .
func Point2XYPoint(p Point, begin time.Time, unit time.Duration) XYPoint {
	return XYPoint{
		X: Timestamp2X(begin, p.Stamp(), unit),
		Y: p.Value(),
	}
}

// Points2XYPoints .
func Points2XYPoints(ps Points, begin time.Time, unit time.Duration) XYPoints {
	xsps := make(XYPoints, 0, len(ps))
	for _, p := range ps {
		xsps = append(xsps, Point2XYPoint(p, begin, unit))
	}

	return xsps
}

// PerTrichotomyFitOp .
type PerTrichotomyFitOp struct {
	MaxBadPer        float64 // 最大坏点比例
	AngleIntervalNum int     // 角度区间分片数
	AErr             float64 // 斜率三分终止误差
	BErr             float64 // 常数三分终止误差
}

// PerTrichotomyFit .
//  对points寻找一条直线: y = a*x + b;
//	该直线要求对于points中拟合最好的 (1 - maxBadPer)% 的数据, 标准差最小;
//  相对于LSFit, 该算法可以避免doc/pics/bad_data_line.png中的情况:
//	  某些错误点对直线拟合结果产生巨大的影响;
//  Impl:
//	 枚举斜率区间, 在该区间中三分斜率a, 然后三分常数b
func PerTrichotomyFit(points XYPoints, op *PerTrichotomyFitOp) (float64, float64, float64) {
	deltaAngle := math.Pi / float64(op.AngleIntervalNum)
	var bestA, bestB float64
	bestSD := math.Inf(1)

	for i := 0; i < op.AngleIntervalNum; i++ {
		leftAngle := deltaAngle * float64(i)
		rightAngle := leftAngle + deltaAngle
		for rightAngle-leftAngle > 0.001 { // 三分求解区间内最优值
			_1_3angle := leftAngle + (rightAngle-leftAngle)/3
			_1_3a := math.Tan(_1_3angle)
			_1_3b := perTrichotomyFitWithA(points, op.MaxBadPer, _1_3a, op.BErr)
			_1_3sd := perLineSD(points, op.MaxBadPer, _1_3a, _1_3b)

			_2_3angle := rightAngle - (rightAngle-leftAngle)/3
			_2_3a := math.Tan(_2_3angle)
			_2_3b := perTrichotomyFitWithA(points, op.MaxBadPer, _2_3a, op.BErr)
			_2_3sd := perLineSD(points, op.MaxBadPer, _2_3a, _2_3b)

			if _1_3sd < _2_3sd {
				rightAngle = _2_3angle
			} else {
				leftAngle = _1_3angle
			}
		}

		a := math.Tan((leftAngle + rightAngle) / 2)
		b := perTrichotomyFitWithA(points, op.MaxBadPer, a, op.BErr)
		sd := perLineSD(points, op.MaxBadPer, a, b)

		if sd < bestSD {
			bestA = a
			bestB = b
			bestSD = sd
		}
	}

	return bestA, bestB, bestSD
}

// perTrichotomyFitWithA .
func perTrichotomyFitWithA(points XYPoints, maxBadPer, a, bErr float64) float64 {
	left := math.Inf(1)
	right := math.Inf(-1)
	for _, p := range points {
		v := p.X * a
		diff := p.Y - v
		left = math.Min(left, diff)
		right = math.Max(right, diff)
	}

	for right-left > bErr {
		_1_3b := left + (right-left)/3
		_1_3sd := perLineSD(points, maxBadPer, a, _1_3b)

		_2_3b := right - (right-left)/3
		_2_3sd := perLineSD(points, maxBadPer, a, _2_3b)

		if _1_3sd < _2_3sd {
			right = _2_3b
		} else {
			left = _1_3b
		}
	}

	return (left + right) / 2
}

/*
perLineSD 根据 y = ax+b 求出points所有点的误差;
	求误差最小的(1 - maxBadPer)的sd;
*/
func perLineSD(points XYPoints, maxBadPer, a, b float64) float64 {
	vars := make([]float64, len(points))
	for i := range points {
		y := a*points[i].X + b
		diff := y - points[i].Y
		vars[i] = diff * diff
	}

	sort.Float64s(vars)
	lim := int(float64(len(points)) * (1 - maxBadPer))
	avg := AVG(vars[:lim])
	return math.Sqrt(avg)
}

// LSFit Least squares
func LSFit(points XYPoints) (a float64, b float64) {
	if len(points) == 0 {
		return 0, 0
	}

	// http://ja.wikipedia.org/wiki/%E6%9C%80%E5%B0%8F%E4%BA%8C%E4%B9%97%E6%B3%95
	n := float64(len(points))
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumXX := 0.0

	for _, p := range points {
		sumX += p.X
		sumY += p.Y
		sumXY += p.X * p.Y
		sumXX += p.X * p.X
	}

	base := (n*sumXX - sumX*sumX)
	a = (n*sumXY - sumX*sumY) / base
	b = (sumXX*sumY - sumXY*sumX) / base
	return a, b
}
