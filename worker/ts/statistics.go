package ts

import (
	"math"
	"sort"
)

// PercentThreshold .
func PercentThreshold(vals []float64, per float64) float64 {
	if len(vals) == 0 {
		return 0
	}

	if per < 0 {
		per = 0
	} else if per > 1 {
		per = 1
	}

	sort.Float64s(vals)
	return vals[int(per*float64(len(vals)))]
}

// PercentValues .
func PercentValues(vals []float64, per float64) []float64 {
	threshold := PercentThreshold(vals, per)
	results := make([]float64, 0, len(vals))
	for _, p := range vals {
		if p <= threshold {
			results = append(results, p)
		}
	}
	return results
}

// Points2Vals convert these points to []float64
func Points2Vals(ps Points) []float64 {
	vals := make([]float64, 0, len(ps))
	for _, p := range ps {
		vals = append(vals, p.Value())
	}
	return vals
}

// AVGPoints .
func AVGPoints(ps Points) float64 {
	return AVG(Points2Vals(ps))
}

// AVG .
func AVG(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	return Sum(vals) / float64(len(vals))
}

// SD calculate standard deviation
func SD(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}

	avg := AVG(vals)
	var diffSum float64
	for _, v := range vals {
		diffSum += (v - avg) * (v - avg)
	}

	return math.Sqrt(diffSum / float64(len(vals)))
}

// SDPoints .
func SDPoints(ps Points) float64 {
	return SD(Points2Vals(ps))
}

// Sum .
func Sum(vals []float64) float64 {
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum
}

// MinKSDThreshold .
//  从大到小剔除vals中的数据, 直到vals中所有的数据都在"k倍标准差"范围内;
//	最多剔除 maxRmPer 百分比的数据;
func MinKSDThreshold(vals []float64, k, maxRmPer float64) (threshold float64) {
	maxRm := int(float64(len(vals)) * maxRmPer)
	rmed := 0
	sort.Float64s(vals)
	for rmed < maxRm {
		avg := AVG(vals)
		sd := SD(vals)
		threshold := avg + k*sd

		last := rmed
		for rmed < maxRm && vals[len(vals)-1] > threshold {
			vals = vals[:len(vals)-1]
			rmed++
		}

		if last == rmed {
			break // no more value can be removed
		}
	}

	if len(vals) == 0 {
		return 0
	}

	return vals[len(vals)-1]
}

// MinKSDValues .
func MinKSDValues(vals []float64, k, maxRmPer float64) []float64 {
	threshold := MinKSDThreshold(vals, k, maxRmPer)
	results := make([]float64, 0, len(vals))
	for _, v := range vals {
		if v <= threshold {
			results = append(results, v)
		}
	}
	return results
}
