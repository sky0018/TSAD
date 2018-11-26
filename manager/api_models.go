package manager

import "time"

// TimeSeries .
type TimeSeries struct {
	TaskName    string     `json:"task_name"`
	DataSource  DataSource `json:"data_source"`
	DerivedAt   time.Time  `json:"derived_at"`
	DerivedHost string     `json:"derived_host"`

	// runtime information
	State          string    `json:"state"`
	Model          string    `json:"model"`
	LastError      string    `json:"last_error"`
	LastErrorStamp time.Time `json:"last_error_stamp"`
	LastDetectedAt time.Time `json:"last_detected_at"`
}

// DataSource .
type DataSource struct {
	Type  string `json:"type"`
	Key   string `json:"key"`
	Extra string `json:"extra"`
}

// TaskMeta .
type TaskMeta struct {
	Name       string     `json:"name"` // primary key
	DataSource DataSource `json:"data_source"`
	Config     string     `json:"config"`
}

// TaskDetail .
type TaskDetail struct {
	TaskMeta
	State          string        `json:"state"`
	Error          string        `json:"error"`
	LastError      string        `json:"last_error"`       // the latest err on the task
	LastErrorStamp time.Time     `json:"last_error_stamp"` // the latest error timestamp
	Timeseries     []*TimeSeries `json:"timeseries"`
	ProcessedBy    string        `json:"processed_by"`
}

// Point .
type Point struct {
	Value float64   `json:"value"`
	Stamp time.Time `json:"stamp"`
}

// ForecastTS .
type ForecastTS struct {
	DataSource DataSource `json:"data_source"`
	Error      string     `json:"error"`
	Observe    []*Point   `json:"observe"`
	Uppert     []*Point   `json:"upper"`
	Lower      []*Point   `json:"lower"`
}
