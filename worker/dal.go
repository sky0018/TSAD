package worker

import (
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
)

var (
	db *gorm.DB
)

func initDAL() error {
	var err error
	db, err = gorm.Open("mysql", config.MysqlDSN)
	if err != nil {
		return fmt.Errorf("open %v err: %v", config.MysqlDSN, err)
	}
	return nil
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

// UpdateDetectorInfo .
func UpdateDetectorInfo(d *Detector) error {
	return db.Save(d).Error
}

// ModelData .
type ModelData struct {
	SrcKey string `gorm:"primary_key;not null;unique_index:uniq_key"`
	Name   string
	Data   string
	Stamp  time.Time
}

// TableName .
func (md ModelData) TableName() string {
	return "tsad_model_data"
}

// SaveModelData .
func SaveModelData(d *ModelData) error {
	return db.Save(d).Error
}

// QueryModelData .
func QueryModelData(key string) (*ModelData, error) {
	var data ModelData
	err := db.Where("src_key=?", key).First(&data).Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}

// DeleteModelData .
func DeleteModelData(key string) error {
	return db.Where("src_key=?", key).Delete(ModelData{}).Error
}
