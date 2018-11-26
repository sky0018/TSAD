package ts

import "time"

// Point represent a point in a TS
type Point interface {
	Stamp() time.Time
	Value() float64
}

type point struct {
	unix  int64
	value float64
}

func (p point) Stamp() time.Time {
	return time.Unix(p.unix, 0)
}

func (p point) Value() float64 {
	return p.value
}

// NewPoint .
func NewPoint(stamp time.Time, val float64) Point {
	return point{stamp.Unix(), val}
}

// Points .
type Points []Point

// Len .
func (p Points) Len() int {
	return len(p)
}

// Less .
func (p Points) Less(i, j int) bool {
	return p[i].Stamp().Before(p[j].Stamp())
}

// Swap .
func (p Points) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

// LeftBinSearch find the max pos that timestamp <= stamp
func (p Points) LeftBinSearch(stamp time.Time) int {
	left := 0
	right := len(p)
	for left < right {
		mid := (left + right) >> 1
		t := p[mid].Stamp()
		if t.Equal(stamp) {
			return mid
		} else if stamp.After(t) {
			left = mid + 1
		} else {
			right = mid
		}
	}

	if left == len(p) {
		left--
	}
	return left
}

// UnknownOrInvalid .
const UnknownOrInvalid = 0

// Attributes attributes for this TS
type Attributes struct {
	Frequency time.Duration // points[i+1].Stamp() - points[i].Stamp()
	Period    time.Duration // period of this TS
}

// TS represent a time-series
type TS interface {
	Attributes() Attributes
	Frequency() time.Duration
	Period() time.Duration
	Points() Points
	GetPoint(stamp time.Time) (Point, bool)
	GetPoints(begin, end time.Time) Points
	Values() []float64
	Begin() time.Time
	End() time.Time
	N() int
	Completed() bool
}

type ts struct {
	attr   Attributes
	points Points
}

// NewTS .
func NewTS(attr Attributes, points Points) TS {
	at := attr
	ps := make(Points, len(points))
	copy(ps, points)
	return ts{at, ps}
}

func (ts ts) GetPoint(stamp time.Time) (Point, bool) {
	idx := ts.points.LeftBinSearch(stamp)
	if ts.points[idx].Stamp().Equal(stamp) {
		return ts.points[idx], true
	}

	return nil, false
}

// return the points in [begin, end]
func (ts ts) GetPoints(begin, end time.Time) Points {
	left := ts.points.LeftBinSearch(begin)
	if !ts.points[left].Stamp().Equal(begin) {
		left++
	}
	right := ts.points.LeftBinSearch(end)
	right++

	return ts.points[left:right]
}

// Attributes .
func (ts ts) Attributes() Attributes {
	return ts.attr
}

// Frequency .
func (ts ts) Frequency() time.Duration {
	return ts.attr.Frequency
}

// Period .
func (ts ts) Period() time.Duration {
	return ts.attr.Period
}

// Points .
func (ts ts) Points() Points {
	points := make(Points, len(ts.points))
	copy(points, ts.points)
	return points
}

// Values .
func (ts ts) Values() []float64 {
	vals := make([]float64, 0, len(ts.points))
	for _, p := range ts.points {
		vals = append(vals, p.Value())
	}
	return vals
}

// Begin beginning of observation
func (ts ts) Begin() time.Time {
	if len(ts.points) == 0 {
		return time.Time{}
	}
	return ts.points[0].Stamp()
}

// End end of observation
func (ts ts) End() time.Time {
	if len(ts.points) == 0 {
		return time.Time{}
	}
	return ts.points[len(ts.points)-1].Stamp()
}

// N number of observed points
func (ts ts) N() int {
	return len(ts.points)
}

// Completed if this TS is completed
// if in each timestamp (Begin + OberveInterval * i) & (0 < i < N),
//  there is a observation point, then this TS is completed;
func (ts ts) Completed() bool {
	for i := 1; i < ts.N(); i++ {
		if ts.points[i].Stamp().Sub(ts.points[i-1].Stamp()) != ts.attr.Frequency {
			return false
		}
	}
	return true
}
