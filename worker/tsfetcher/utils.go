package tsfetcher

import (
	"time"

	"code.byted.org/microservice/tsad/worker/ts"
)

var (
	maxGap = time.Minute * 15
)

func complete(points ts.Points, freq time.Duration) ts.Points {
	if len(points) < 2 {
		return points
	}

	results := make(ts.Points, 0, len(points))
	results = append(results, points[0])
	for i := 1; i < len(points); i++ {
		gap := points[i].Stamp().Sub(points[i-1].Stamp())
		if gap <= freq || gap > maxGap {
			results = append(results, points[i])
		} else {
			diff := points[i].Value() - points[i-1].Value()
			n := gap / freq
			delta := diff / float64(n)

			for j := 1; j < int(n); j++ {
				results = append(results, ts.NewPoint(
					points[i-1].Stamp().Add(time.Duration(j)*freq),
					points[i-1].Value()+float64(j)*delta,
				))
			}

			results = append(results, points[i])
		}
	}

	return results
}
