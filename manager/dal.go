package manager

import (
	"time"

	"github.com/jinzhu/gorm"
)

var (
	dbRead  *gorm.DB
	dbWrite *gorm.DB

	fooTime = time.Unix(0, 0).Add(time.Hour * 24 * 365 * 30)
)

func initDAL() error {
	var err error
	dbRead, err = gorm.Open("mysql", config.DSNRead)
	if err != nil {
		return err
	}
	dbWrite, err = gorm.Open("mysql", config.DSNWrite)
	if err != nil {
		return err
	}

	dbWrite.AutoMigrate(&Task{})
	dbWrite.AutoMigrate(&Detector{})
	return nil
}

// GetTasks .
func GetTasks() ([]*Task, error) {
	var ts []*Task
	err := dbRead.Find(&ts).Error
	return ts, err
}

// GetTaskByName .
func GetTaskByName(name string) (*Task, error) {
	var task Task
	err := dbRead.Where("`name`=?", name).First(&task).Error
	return &task, err
}

// UpdateTaskState .
func UpdateTaskState(name, state string) error {
	dbWrite.UpdateColumn()
	return dbWrite.Model(&Task{}).Where("`name`=?", name).UpdateColumn("state", state).Error
}

// UpdateTaskByName .
func UpdateTaskByName(oldName string, t *Task) error {
	return dbWrite.Model(t).Where("`name`=?", oldName).Updates(t).Error
}

// InsertTask .
func InsertTask(name, src, conf string) error {
	t := &Task{
		LockExpiration: fooTime,
		Name:           name,
		DataSource:     src,
		Config:         conf,
		State:          TaskRunning,
	}
	return dbWrite.Create(t).Error
}

// GetDetectors .
func GetDetectors() ([]*Detector, error) {
	var ds []*Detector
	err := dbRead.Find(&ds).Error
	return ds, err
}

// UpsertDetector .
func UpsertDetector(d *Detector) error {
	return dbWrite.Save(d).Error
}
