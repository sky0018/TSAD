package tspreprocess

import (
	"time"

	"code.byted.org/microservice/tsad/worker/ts"
)

// OneNNOutlierFilter .
func OneNNOutlierFilter(data ts.TS) ts.TS {
	totPoints := data.Points()
	totXYPoints := make(ts.XYPoints, 0, len(totPoints))

	begin := 0
	maxLen := 1440
	for begin < len(totPoints) {
		end := begin + maxLen
		if end > len(totPoints) {
			end = len(totPoints)
		}

		points := totPoints[begin:end]
		xypoints := make(ts.XYPoints, len(points))
		for i, p := range points {
			xypoints[i].X = float64(p.Stamp().Unix() - data.Begin().Unix())
			xypoints[i].Y = p.Value()
		}

		cleaned := ts.OneNNOutlierFilter(xypoints, 0.1)
		cleaned.SortByX()
		totXYPoints = append(totXYPoints, cleaned...)

		begin = end
	}

	cpoints := make(ts.Points, 0, len(totXYPoints))
	for _, p := range totXYPoints {
		cpoints = append(cpoints, ts.NewPoint(time.Unix(data.Begin().Unix()+int64(p.X), 0), p.Y))
	}

	return ts.NewTS(data.Attributes(), cpoints)
}
