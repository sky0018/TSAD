package manager

import "time"

const (
	// TaskStopped .
	TaskStopped = "stopped"
	// TaskRunning .
	TaskRunning = "running"
)

// Task .
type Task struct {
	Name           string `gorm:"primary_key;not null;unique_index:uniq_name"`
	State          string
	DataSource     string
	Config         string
	ProcessedBy    string
	LockExpiration time.Time
}

// TableName .
func (t Task) TableName() string {
	return "tsad_tasks"
}

// Detector .
type Detector struct {
	Host      string    `json:"host" gorm:"primary_key;not null;unique_index:uniq_host"`
	NumTasks  int       `json:"num_tasks"` // number of tasks
	HeartBeat time.Time `json:"heart_beat"`
}

// TableName .
func (d Detector) TableName() string {
	return "tsad_detectors"
}
