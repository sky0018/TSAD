package ts

// LastValueComplete .
func LastValueComplete(data TS) TS {
	points := make([]Point, 0, 1024)
	lastVal := 0.0
	for now := data.Begin(); !now.After(data.End()); now = now.Add(data.Frequency()) {
		p, ok := data.GetPoint(now)
		if ok {
			points = append(points, p)
			lastVal = p.Value()
		} else {
			points = append(points, NewPoint(now, lastVal))
		}
	}

	return NewTS(data.Attributes(), points)
}
