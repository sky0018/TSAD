package ts

// import (
// 	"fmt"
// 	"math"
// 	"math/rand"
// 	"testing"
// 	"time"

// 	"code.byted.org/microservice/tsad/testtools"
// )

// func TestMovingLinearClean(t *testing.T) {
// 	begin := time.Now()
// 	freq := time.Second * 30
// 	points := testtools.GenPoints(begin,
// 		begin.Add(time.Hour*10),
// 		freq, &testtools.UniRandGener{Min: 0, Max: 0})

// 	// insert some bad points
// 	n := len(points)
// 	type badCase struct {
// 		index int
// 		value float64
// 	}
// 	cases := make([]*badCase, 0, 100)
// 	for i := 0; i < n/1000; i++ {
// 		index := rand.Intn(n)
// 		points[index].Value = 10000
// 		cases = append(cases, &badCase{i, 10000})
// 	}

// 	tsdata := NewTS(Attributes{
// 		Frequency: freq,
// 	}, points)
// 	cleaned, err := MovingLinearClean(tsdata, time.Minute*5)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	cleanedPoints := cleaned.Points()
// 	for _, bad := range cases {
// 		if math.Abs(cleanedPoints[bad.index].Value-bad.value) < 0.001 {
// 			t.Fatal("err")
// 		}
// 		if cleanedPoints[bad.index].Value > 1 {
// 			fmt.Println(bad.index, bad.value)
// 			fmt.Println(cleanedPoints[bad.index].Value)
// 			t.Fatal("err")
// 		}
// 	}
// }
