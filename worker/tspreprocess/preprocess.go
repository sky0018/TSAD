package tspreprocess

import (
	"fmt"
	"time"

	"code.byted.org/microservice/tsad/worker/ts"
)

// BasicTSCheck 检查该ts的基本属性;
//  同时判断是否过短或者过于稀疏;
func BasicTSCheck(data ts.TS) error {
	obDuration := data.End().Sub(data.Begin())
	if obDuration < time.Hour*24*2 { // 观测时常必须大于2天
		return fmt.Errorf("observation duration is too short: %v", obDuration)
	}

	if data.Frequency() == ts.UnknownOrInvalid { // 没有观测频率
		return fmt.Errorf("no frequency")
	}

	n := data.N()
	total := obDuration / data.Frequency()
	if float64(n)/float64(total) < 0.8 {
		return fmt.Errorf("time-series is too sparse / 时序过于稀疏, 时间跨度=%v, 观测点数=%v, 需要点数=%v", obDuration, n, int(total))
	}

	return nil
}

// Preprocess .
func Preprocess(data ts.TS) (ts.TS, error) {
	if err := BasicTSCheck(data); err != nil {
		return nil, err
	}

	// // if no latest data, we think this ts is stopped
	// if data.End().Add(time.Minute * 10).Before(time.Now()) {
	// 	return nil, fmt.Errorf("no latest data")
	// }

	data = OneNNOutlierFilter(data)
	if err := BasicTSCheck(data); err != nil {
		return nil, err
	}

	return data, nil
}
