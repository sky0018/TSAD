package testtools

import (
	"math"
	"math/rand"
	"sort"
	"time"

	"code.byted.org/microservice/tsad/ts"
)

// Stamp2X convert a timestamp to a value on X
type Stamp2X func(stamp time.Time) float64

// LineGener y = Ax + B
type LineGener struct {
	A       float64
	B       float64
	Stamp2X Stamp2X
}

// Gen .
func (lg *LineGener) Gen(stamp time.Time) float64 {
	x := lg.Stamp2X(stamp)
	return lg.A*x + lg.B
}

// SinGener .
type SinGener struct {
	Stamp2X Stamp2X
}

// Gen .
func (sg *SinGener) Gen(stamp time.Time) float64 {
	x := sg.Stamp2X(stamp)
	return math.Sin(x)
}

// UniRandGener uniform distribution random generator
type UniRandGener struct {
	Min float64
	Max float64
}

// Gen .
func (urg *UniRandGener) Gen(stamp time.Time) float64 {
	k := urg.Max - urg.Min
	r := rand.Float64() * k
	return r + urg.Min
}

// Generator .
type Generator interface {
	Gen(stamp time.Time) float64
}

// GenPoints generate a TS
func GenPoints(begin, end time.Time, freq time.Duration, g Generator) ts.Points {
	var points ts.Points
	for stamp := begin; !stamp.After(end); stamp = stamp.Add(freq) {
		points = append(points, ts.NewPoint(
			stamp,
			g.Gen(stamp)))
	}
	return points
}

// AddTS .
func AddTS(points1, points2 ts.Points) ts.Points {
	pointMap := make(map[time.Time]float64)
	for _, p := range points1 {
		pointMap[p.Stamp()] += p.Value()
	}
	for _, p := range points2 {
		pointMap[p.Stamp()] += p.Value()
	}

	var points ts.Points
	for t, v := range pointMap {
		points = append(points, ts.NewPoint(
			t, v,
		))
	}

	sort.Sort(points)
	return points
}
