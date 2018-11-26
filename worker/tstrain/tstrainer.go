package tstrain

import (
	"fmt"
	"os"
	"time"

	"code.byted.org/microservice/tsad/worker/detector"
	"code.byted.org/microservice/tsad/worker/ts"
	"code.byted.org/microservice/tsad/worker/tstrain/tsmodels"
)

var creator map[string]func() detector.TSModel

func init() {
	creator = make(map[string]func() detector.TSModel)

	// creator["Period3SigmaModel"] = func() TSModel {
	// 	return tsmodels.NewPeriod3SigmaModel()
	// }

	// creator["LineModel"] = func() TSModel {
	// 	return tsmodels.NewLineModel()
	// }

	creator["DcmpLineModel"] = func() detector.TSModel {
		return tsmodels.NewDcmpLineModel()
	}
}

// Train .
func Train(data ts.TS, adapter detector.ModelAdapter) (detector.TSModel, error) {
	cs := make([]func() detector.TSModel, 0, 20)
	for _, c := range creator {
		cs = append(cs, c)
	}

	ms := make([]detector.TSModel, 0, len(cs))
	for _, c := range cs {
		m := c()
		begin := time.Now()
		err := m.Train(data, adapter)
		if err == nil {
			fmt.Printf("train %v cost %v\n", m.Name(), time.Since(begin))
			ms = append(ms, m)
		} else {
			fmt.Fprintf(os.Stderr, "train %v err: %v", m.Name(), err)
		}
	}

	if len(ms) == 0 {
		return nil, fmt.Errorf("no model can be trained for these data")
	}

	best, err := YahooEgadsPicker(data, ms)
	if err != nil {
		return nil, fmt.Errorf("YahooEgadsPicker can't pick the best model, err: %v", err)
	}

	return best, nil
}

// Recover .
func Recover(name string, data []byte) (detector.TSModel, error) {
	c, ok := creator[name]
	if !ok {
		return nil, fmt.Errorf("no model name: %v", name)
	}

	m := c()
	if err := m.Recover(data); err != nil {
		return nil, fmt.Errorf("recover model %v err: %v", name, err)
	}

	return m, nil
}
