package utils

import (
	"context"
	"fmt"
	"time"

	"code.byted.org/microservice/tsad/testtools"
	"code.byted.org/microservice/tsad/ts"
	"code.byted.org/microservice/tsad/tsfetcher"
	"code.byted.org/microservice/tsad/tspreprocessor"
	"code.byted.org/microservice/tsad/tstrainer"
)

// TestCleanWithMetrics .
func TestCleanWithMetrics(metrics string, begin, end time.Time) {
	fmt.Println("fetching...")
	fetcher, err := tsfetcher.NewTSDBFetcher("http://metrics.byted.org/api/query",
		10, time.Second*10)
	if err != nil {
		panic(err)
	}

	tsdata, err := fetcher.Fetch(context.Background(),
		&tsfetcher.Source{
			Type:  tsfetcher.SourceTSDB,
			Key:   metrics,
			Extra: ""},
		begin, end)
	if err != nil {
		panic(err)
	}

	if err := testtools.Points2CSVFile("./src.csv", tsdata.Points()); err != nil {
		panic(err)
	}

	fmt.Println("cleaning...")
	cleaned, err := tspreprocessor.Preprcess(tsdata)
	if err != nil {
		panic(err)
	}

	if err := testtools.Points2CSVFile("./cleaned.csv", cleaned.Points()); err != nil {
		panic(err)
	}
}

// TestModelWithMetrics .
func TestModelWithMetrics(m tstrainer.TSModel,
	metrics string, begin, end time.Time) {

	// fmt.Println("fetching...")
	// fetcher, err := tsfetcher.NewTSDBFetcher("http://metrics.byted.org/api/query",
	// 	10, time.Second*10)
	// if err != nil {
	// 	panic(err)
	// }

	// tsdata, err := fetcher.Fetch(context.Background(),
	// 	&tsfetcher.Source{
	// 		Type:  tsfetcher.SourceTSDB,
	// 		Key:   metrics,
	// 		Extra: ""},
	// 	begin, end)
	// if err != nil {
	// 	panic(err)
	// }

	data, err := testtools.CSVFile2Points("/Users/zhangyuanjia/Work/src/lab/yyy/src.csv")
	if err != nil {
		panic(err)
	}
	tsdata := ts.NewTS(tsfetcher.TSDBTSAttr, data)

	if err := testtools.Points2CSVFile("./src.csv", tsdata.Points()); err != nil {
		panic(err)
	}

	fmt.Println("cleaning...")
	cleaned, err := tspreprocessor.Preprcess(tsdata)
	if err != nil {
		panic(err)
	}

	if err := testtools.Points2CSVFile("./cleaned.csv", cleaned.Points()); err != nil {
		panic(err)
	}

	fmt.Println("training...")
	if err := m.Train(cleaned); err != nil {
		panic(err)
	}

	fmt.Println("forcasting...")
	lower := make(ts.Points, 0, tsdata.N())
	upper := make(ts.Points, 0, tsdata.N())
	for _, p := range tsdata.Points() {
		l, u := m.ForecastInterval(p.Stamp())
		lower = append(lower, ts.NewPoint(p.Stamp(), l))
		upper = append(upper, ts.NewPoint(p.Stamp(), u))
	}

	if err := testtools.Points2CSVFile("./lower.csv", lower); err != nil {
		panic(err)
	}
	if err := testtools.Points2CSVFile("./upper.csv", upper); err != nil {
		panic(err)
	}
}
