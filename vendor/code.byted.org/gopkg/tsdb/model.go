package tsdb

import "sort"

// Tags .
type Tags map[string]string

// DPS .
type DPS map[string]float32

// AVG .
func (d DPS) AVG() float32 {
	if len(d) == 0 {
		return 0
	}

	var sum float32
	for _, f := range d {
		sum += f
	}
	return sum / float32(len(d))
}

// PCT .
func (d DPS) PCT(per float32) float32 {
	if len(d) == 0 {
		return 0
	}

	nums := make([]float64, 0, len(d))
	for _, v := range d {
		nums = append(nums, float64(v))
	}

	sort.Float64s(nums)
	index := int(float32(len(nums)) * per / 100)
	return float32(nums[index])
}

// AggregateTags .
type AggregateTags []string

// RespModel .
type RespModel struct {
	Metric        string        `json:"metric"`
	Tags          Tags          `json:"tags"`
	AggregateTags AggregateTags `json:"aggregateTags"`
	DPS           DPS           `json:"dps"`
}
