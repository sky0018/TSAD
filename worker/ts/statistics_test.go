package ts

import (
	"math/rand"
	"testing"
)

func TestMinKSDValues1(t *testing.T) {
	var vals []float64
	n := 1000
	for i := 0; i < n; i++ {
		vals = append(vals, rand.Float64())
	}
	vals = append(vals, 10000.0)
	vals = append(vals, 20000.0)
	vals = append(vals, 30000.0)

	threshold := MinKSDThreshold(vals, 3, 0.2)
	if threshold >= 10000 {
		t.Fatal("err")
	}

	filtered := MinKSDValues(vals, 3, 0.2)
	if len(filtered) != n {
		t.Fatal("err")
	}

	for _, v := range filtered {
		if v > 1 {
			t.Fatal("err")
		}
	}
}
