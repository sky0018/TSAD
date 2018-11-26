package ts

import (
	"testing"
	"time"
)

func TestPeriodShift(t *testing.T) {
	begin := time.Now()
	period := time.Hour

	if PeriodShift(begin.Add(time.Second), begin, period) != time.Second {
		t.Fatal("error")
	}

	if PeriodShift(begin.Add(-time.Second), begin, period) != (period - time.Second) {
		t.Fatal("error")
	}
}
