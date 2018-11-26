package tsmodels

import (
	"testing"
	"time"
)

func TestDcmpLineModelData(t *testing.T) {
	m := &DcmpLineModel{
		Freq:   1,
		Begin:  time.Now(),
		Period: time.Hour,
		PeriodTrend: map[time.Duration]float64{
			time.Hour:   1.1,
			time.Minute: 2.2,
		},
		PeriodSeaon: map[time.Duration]float64{
			time.Hour:   341.123123,
			time.Minute: 2132.22,
		},
		RandomAVG:    233.3,
		RandomSD:     1234.43,
		LowerAdapter: 23.23,
		UpperAdapter: 98,
	}

	data, err := m.ModelData()
	if err != nil {
		t.Fatal(err)
	}

	m1 := &DcmpLineModel{}
	err = m1.Recover(data)
	if err != nil {
		t.Fatal(err)
	}

	ok := func(flag bool) {
		if flag == false {
			t.Fatal(err)
		}
	}

	ok(m1.Freq == m.Freq)
	ok(m1.Begin == m.Begin)
	ok(m1.Period == m.Period)
	ok(m1.RandomAVG == m.RandomAVG)
	ok(m1.RandomSD == m.RandomSD)
	ok(m1.LowerAdapter == m.LowerAdapter)
	ok(m1.UpperAdapter == m.UpperAdapter)
	for k := range m1.PeriodTrend {
		ok(m1.PeriodTrend[k] == m.PeriodTrend[k])
	}
	for k := range m1.PeriodSeaon {
		ok(m1.PeriodSeaon[k] == m.PeriodSeaon[k])
	}
}
