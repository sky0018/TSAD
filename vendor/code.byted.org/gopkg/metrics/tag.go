package metrics

import "strings"

type T struct {
	Name  string
	Value string
}

func (t T) less(o T) bool {
	if c := strings.Compare(t.Name, o.Name); c != 0 {
		return c < 0
	}
	return t.Value < o.Value
}

func SortTags(ss []T) {
	for i := 1; i < len(ss); i++ {
		for j := i; j > 0 && ss[j].less(ss[j-1]); j-- {
			ss[j], ss[j-1] = ss[j-1], ss[j]
		}
	}
}

func TagsEqual(ts1, ts2 []T) bool {
	if len(ts1) != len(ts2) {
		return false
	}
	for i := range ts1 {
		if ts1[i] != ts2[i] {
			return false
		}
	}
	return true
}
