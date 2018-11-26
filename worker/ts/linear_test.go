package ts

import (
	"math"
	"testing"
)

func TestLSFit(t *testing.T) {
	var xys XYPoints
	a := 0.0
	b := 3.432
	for i := 0; i < 100; i++ {
		xys = append(xys, XYPoint{
			X: float64(i),
			Y: a*float64(i) + b,
		})
	}

	fa, fb := LSFit(xys)
	if math.Abs(a-fa) > 0.001 {
		t.Fatal("err")
	}
	if math.Abs(b-fb) > 0.001 {
		t.Fatal("err")
	}
}

func TestPerTrichotomyFit(t *testing.T) {
	var xys XYPoints
	a := 23.333
	b := 32.33
	for i := 0; i < 100; i++ {
		xys = append(xys, XYPoint{
			X: float64(i),
			Y: a*float64(i) + b,
		})
	}

	xys[0].Y = 100000
	xys[10].Y = -11231
	xys[33].Y = 837432

	fa, fb, _ := PerTrichotomyFit(xys, &PerTrichotomyFitOp{
		MaxBadPer:        0.1,
		AngleIntervalNum: 100,
		AErr:             0.001,
		BErr:             0.001,
	})
	if math.Abs((fa-a)/a) > 0.1 {
		t.Fatal("err")
	}
	if math.Abs((fb-b)/b) > 0.1 {
		t.Fatal("err")
	}
}
