package tstrain

import (
	"fmt"
	"math"

	"code.byted.org/microservice/tsad/worker/detector"
	"code.byted.org/microservice/tsad/worker/ts"
)

// TSModelPicker .
type TSModelPicker func([]detector.TSModel) (best detector.TSModel, err error)

type yahooEgadsMetric struct {
	bias float64
	mad  float64
	mape float64
	mse  float64
	sae  float64
}

func yahooEgadsCalMetric(srcTS ts.TS, m detector.TSModel) *yahooEgadsMetric {
	var sumErr, sumAbsErr, sumAbsPercentErr, sumErrSquared float64
	for _, observ := range srcTS.Points() {
		forecast := m.Forecast(observ.Stamp())
		deltaErr := forecast - observ.Value()

		sumErr += deltaErr
		sumAbsErr += math.Abs(deltaErr)
		sumAbsPercentErr += math.Abs(deltaErr / observ.Value())
		sumErrSquared += deltaErr * deltaErr
	}

	n := float64(len(srcTS.Points()))
	return &yahooEgadsMetric{
		bias: sumErr / n,
		mad:  sumAbsErr / n,
		mape: sumAbsPercentErr / n,
		mse:  sumErrSquared / n,
		sae:  sumAbsErr,
	}
}

func yahooEgadsBetterThan(m1, m2 *yahooEgadsMetric) bool {
	tolerance := 0.00000001
	calScore := func(delta1, delta2 float64) int {
		if math.IsNaN(delta1) || math.IsNaN(delta2) {
			return 0
		}

		diffAbs := math.Abs(delta2) - math.Abs(delta1)
		if math.Abs(diffAbs) <= tolerance {
			return 0
		}
		if diffAbs > 0 {
			return 1
		}
		return -1
	}

	var score int
	score += calScore(m1.bias, m2.bias)
	score += calScore(m1.mad, m2.mad)
	score += calScore(m1.mape, m2.mape)
	score += calScore(m1.mse, m2.mse)
	score += calScore(m1.sae, m2.sae)

	if score == 0 {
		mapeDiff := m1.mape - m2.mape
		diff := m1.bias - m2.bias + m1.mad - m2.mad + m1.mse - m2.mse + m1.sae - m2.sae
		if !math.IsNaN(mapeDiff) {
			diff += mapeDiff
		}

		return diff < 0
	}

	return score > 0

}

// YahooEgadsPicker .
func YahooEgadsPicker(src ts.TS, ms []detector.TSModel) (detector.TSModel, error) {
	if len(ms) == 0 {
		return nil, fmt.Errorf("empty model list")
	}

	metrics := make([]*yahooEgadsMetric, 0, len(ms))
	for _, m := range ms {
		metrics = append(metrics, yahooEgadsCalMetric(src, m))
	}

	bestID := 0
	for i := 1; i < len(ms); i++ {
		if yahooEgadsBetterThan(metrics[i], metrics[bestID]) {
			bestID = i
		}
	}

	return ms[bestID], nil
}
