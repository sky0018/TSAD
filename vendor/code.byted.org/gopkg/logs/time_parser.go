package logs

import (
	"time"
)

// 2016 ~ 2020
var (
	months = []string{
		"2016-01-01 00:00:00", "2016-02-01 00:00:00", "2016-03-01 00:00:00",
		"2016-04-01 00:00:00", "2016-05-01 00:00:00", "2016-06-01 00:00:00",
		"2016-07-01 00:00:00", "2016-08-01 00:00:00", "2016-09-01 00:00:00",
		"2016-10-01 00:00:00", "2016-11-01 00:00:00", "2016-12-01 00:00:00",

		"2017-01-01 00:00:00", "2017-02-01 00:00:00", "2017-03-01 00:00:00",
		"2017-04-01 00:00:00", "2017-05-01 00:00:00", "2017-06-01 00:00:00",
		"2017-07-01 00:00:00", "2017-08-01 00:00:00", "2017-09-01 00:00:00",
		"2017-10-01 00:00:00", "2017-11-01 00:00:00", "2017-12-01 00:00:00",

		"2018-01-01 00:00:00", "2018-02-01 00:00:00", "2018-03-01 00:00:00",
		"2018-04-01 00:00:00", "2018-05-01 00:00:00", "2018-06-01 00:00:00",
		"2018-07-01 00:00:00", "2018-08-01 00:00:00", "2018-09-01 00:00:00",
		"2018-10-01 00:00:00", "2018-11-01 00:00:00", "2018-12-01 00:00:00",

		"2019-01-01 00:00:00", "2019-02-01 00:00:00", "2019-03-01 00:00:00",
		"2019-04-01 00:00:00", "2019-05-01 00:00:00", "2019-06-01 00:00:00",
		"2019-07-01 00:00:00", "2019-08-01 00:00:00", "2019-09-01 00:00:00",
		"2019-10-01 00:00:00", "2019-11-01 00:00:00", "2019-12-01 00:00:00",

		"2020-01-01 00:00:00", "2020-02-01 00:00:00", "2020-03-01 00:00:00",
		"2020-04-01 00:00:00", "2020-05-01 00:00:00", "2020-06-01 00:00:00",
		"2020-07-01 00:00:00", "2020-08-01 00:00:00", "2020-09-01 00:00:00",
		"2020-10-01 00:00:00", "2020-11-01 00:00:00", "2020-12-01 00:00:00",
	}
	monthsNanoSeconds = make([]int64, 12*5)
)

const (
	MillisecondPerDay    = 24 * 60 * 60 * 1000
	MillisecondPerHour   = 60 * 60 * 1000
	MillisecondPerMinute = 60 * 1000
	MillisecondPerSecond = 1000
)

func init() {
	for i, month := range months {
		t, _ := time.ParseInLocation("2006-01-02 15:04:05", month, time.Local)
		monthsNanoSeconds[i] = t.UnixNano()
	}
}

func timeDate(t time.Time) [23]byte {
	var val [23]byte
	idx := findNano(t.UnixNano())
	date := months[idx]
	for i := 0; i < 8; i++ {
		val[i] = byte(date[i])
	}
	dur := int((t.UnixNano() - monthsNanoSeconds[idx]) / 1e6)
	day := dur / MillisecondPerDay
	val[8], val[9], val[10] = byte((day+1)/10+48), byte((day+1)-(day+1)/10*10+48), ' '
	hour := (dur - day*MillisecondPerDay) / MillisecondPerHour
	val[11], val[12], val[13] = byte(hour/10+48), byte(hour-hour/10*10+48), ':'
	minute := (dur - day*MillisecondPerDay - hour*MillisecondPerHour) / MillisecondPerMinute
	val[14], val[15], val[16] = byte(minute/10+48), byte(minute-minute/10*10+48), ':'
	second := (dur - day*MillisecondPerDay - hour*MillisecondPerHour - minute*MillisecondPerMinute) / MillisecondPerSecond
	val[17], val[18], val[19] = byte(second/10+48), byte(second-second/10*10+48), ','
	mils := dur - day*MillisecondPerDay - hour*MillisecondPerHour - minute*MillisecondPerMinute - second*MillisecondPerSecond
	val[20], val[21], val[22] = byte(mils/100+48), byte((mils-mils/100*100)/10+48), byte(mils-mils/10*10+48)
	return val
}

func findNano(nS int64) int {
	for i, nanoSecond := range monthsNanoSeconds {
		if nS < nanoSecond {
			if i == 0 {
				return 0
			}
			return i - 1
		}
	}
	return 0
}
