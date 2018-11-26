package ts

import (
	"fmt"
	"math"
	"testing"
)

func TestInfMin(t *testing.T) {
	a := 1.5
	m := math.Min(a, math.Inf(1))
	if m != a {
		t.Fatal("err")
	}
}

func TestOneNNOutlierFilter(t *testing.T) {
	n := 2000
	points := make(XYPoints, n)
	for i := 0; i < n; i++ {
		points[i].X = float64(i) * 0.01
		points[i].Y = math.Sin(points[i].X)
	}

	// create bad point
	points[23].Y = 2130
	points[54].Y = -123213
	points[223].Y = 399821
	points[123].Y = -21323
	points[1024].Y = -3213213
	points[1834].Y = 923
	bad := 6

	filterd := OneNNOutlierFilter(points, 0.1)
	if len(filterd) != n-bad {
		fmt.Println(">> ", len(filterd))
		t.Fatal("err")
	}

	for i := range filterd {
		if math.Abs(filterd[i].Y) > 1 {
			t.Fatal("err")
		}
	}
}
