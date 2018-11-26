package tsfetcher

import (
	"fmt"
	"testing"
	"time"

	"code.byted.org/microservice/tsad/worker/ts"
)

func TestComplete0(t *testing.T) {
	begin := time.Now()
	points := make(ts.Points, 0, 100)
	points = append(points, ts.NewPoint(begin, 0))
	points = append(points, ts.NewPoint(begin.Add(time.Minute*5), 20))

	points = complete(points, time.Second*30)
	for _, p := range points {
		fmt.Println(p.Stamp(), p.Value())
	}
}

func TestComplete1(t *testing.T) {
	begin := time.Now()
	points := make(ts.Points, 0, 100)
	points = append(points, ts.NewPoint(begin, 0))
	points = append(points, ts.NewPoint(begin.Add(time.Minute*10), 20))

	points = complete(points, time.Second*30)
	for _, p := range points {
		fmt.Println(p.Stamp(), p.Value())
	}
}
