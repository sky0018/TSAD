package main

import (
	"fmt"
	"math/rand"
	"os"

	"code.byted.org/microservice/tsad/testtools"
	"code.byted.org/microservice/tsad/ts"
)

func genLinePointsCSV(csvfile, argfile string, n int,
	badPer, minErr, maxErr float64) error {
	points := make(ts.XYPoints, n)
	a := rand.Float64() * 100
	b := rand.Float64() * 1000

	f, err := os.OpenFile(argfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(fmt.Sprintf("a: %v, b: %v, badPer: %v", a, b, badPer)); err != nil {
		return err
	}

	for i := 0; i < n; i++ {
		points[i].X = float64(i)
		points[i].Y = a*(points[i].X) + b + rand.Float64()*(maxErr-minErr) + minErr
	}

	badCases := int(float64(n) * badPer)
	for i := 0; i < badCases; i++ {
		index := rand.Intn(n)
		points[index].Y += 1e9
	}

	return testtools.XYPoints2CSVFile(csvfile, points)
}

func fitLine(input, output string) error {
	points, err := testtools.CSVFile2XYPoints(input)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}

	a, b := ts.LSFit(points)
	if _, err := f.WriteString(fmt.Sprintf("LSFit: %v %v\n", a, b)); err != nil {
		return err
	}

	a, b, _ = ts.PerTrichotomyFit(points, &ts.PerTrichotomyFitOp{
		MaxBadPer:        0.1,
		AngleIntervalNum: 100,
		AErr:             0.001,
		BErr:             0.001,
	})
	if _, err := f.WriteString(fmt.Sprintf("PerTrichotomyFit: %v %v\n", a, b)); err != nil {
		return err
	}
	return nil
}

func main() {
	if err := genLinePointsCSV("input1.csv", "arg1", 5, 0, 0, 0); err != nil {
		panic(err)
	}
	if err := genLinePointsCSV("input2.csv", "arg2", 10, 0, -1, 1); err != nil {
		panic(err)
	}
	if err := genLinePointsCSV("input3.csv", "arg3", 20, 0, -2, 2); err != nil {
		panic(err)
	}
	if err := genLinePointsCSV("input4.csv", "arg4", 100, 0, -10, 10); err != nil {
		panic(err)
	}

	if err := fitLine("input1.csv", "output1"); err != nil {
		panic(err)
	}
	if err := fitLine("input2.csv", "output2"); err != nil {
		panic(err)
	}
	if err := fitLine("input3.csv", "output3"); err != nil {
		panic(err)
	}
	if err := fitLine("input4.csv", "output4"); err != nil {
		panic(err)
	}

	if err := genLinePointsCSV("input5.csv", "arg5", 20, 0.1, -10, 10); err != nil {
		panic(err)
	}
	if err := genLinePointsCSV("input6.csv", "arg6", 100, 0.1, -10, 10); err != nil {
		panic(err)
	}
	if err := fitLine("input5.csv", "output5"); err != nil {
		panic(err)
	}
	if err := fitLine("input6.csv", "output6"); err != nil {
		panic(err)
	}
}
