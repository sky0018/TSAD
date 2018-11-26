package testtools

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"code.byted.org/microservice/tsad/ts"
)

const (
	pointsHeader   = "timestamp,value"
	xypointsHeader = "x,y"
)

// CSVFile2Points read this csv file and convert it to TS
//  format:
// 	 timestamp(unix second), value
func CSVFile2Points(fpath string) (ts.Points, error) {
	f, err := os.Open(fpath)
	if err != nil {
		return nil, fmt.Errorf("open file err: %v", err)
	}

	r := csv.NewReader(f)
	var points ts.Points
	first := true
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("read file err: %v", err)
		}

		if first {
			first = false
			continue
		}

		if len(record) != 2 {
			return nil, fmt.Errorf("invalid time-series csv file")
		}

		unixSec, err := strconv.Atoi(record[0])
		if err != nil {
			return nil, fmt.Errorf("invalid unix second number in the first column: %v", record[0])
		}
		stamp := time.Unix(int64(unixSec), 0)

		val, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid value in the second column: %v", record[1])
		}

		points = append(points, ts.NewPoint(stamp, val))
	}

	return points, nil
}

// Points2CSVFile write this TS to a csv file
func Points2CSVFile(fpath string, points ts.Points) error {
	f, err := os.OpenFile(fpath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("open file err: %v", err)
	}

	if _, err := f.WriteString(pointsHeader + "\n"); err != nil {
		return fmt.Errorf("write file: %v, err: %v", fpath, err)
	}
	for _, p := range points {
		_, err := f.WriteString(fmt.Sprintf("%v,%v\n", p.Stamp().Unix(), p.Value()))
		if err != nil {
			return fmt.Errorf("write file: %v, err: %v", fpath, err)
		}
	}

	return nil
}

// XYPoints2CSVFile .
func XYPoints2CSVFile(fpath string, points ts.XYPoints) error {
	f, err := os.OpenFile(fpath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("open file err: %v", err)
	}

	if _, err := f.WriteString(xypointsHeader + "\n"); err != nil {
		return fmt.Errorf("write file: %v, err: %v", fpath, err)
	}
	for _, p := range points {
		_, err := f.WriteString(fmt.Sprintf("%v,%v\n", p.X, p.Y))
		if err != nil {
			return fmt.Errorf("write file: %v, err: %v", fpath, err)
		}
	}

	return nil
}

// CSVFile2XYPoints read this csv file and convert it to TS
//  format:
// 	 x, y
func CSVFile2XYPoints(fpath string) (ts.XYPoints, error) {
	f, err := os.Open(fpath)
	if err != nil {
		return nil, fmt.Errorf("open file err: %v", err)
	}

	r := csv.NewReader(f)
	var points ts.XYPoints
	first := true
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("read file err: %v", err)
		}

		if first {
			first = false
			continue
		}

		if len(record) != 2 {
			return nil, fmt.Errorf("invalid time-series csv file")
		}

		x, err := strconv.ParseFloat(record[0], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid float64 in the first column: %v", record[0])
		}

		y, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid float64 in the second column: %v", record[1])
		}

		points = append(points, ts.XYPoint{X: x, Y: y})
	}

	return points, nil
}
