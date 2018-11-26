package tsfetcher

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"code.byted.org/microservice/tsad/worker/ts"
	"github.com/jinzhu/gorm"
)

// MiddleStorer .
type MiddleStorer interface {
	Fetch(src Source) (ts.Points, error)
	Store(src Source, ps ts.Points) error
}

type mysqlTS struct {
	Key    string `gorm:"not null;column:src_key"` // hash of src
	Points string `gorm:"not null;column:points"`  // TS's Points, json format
}

// gorm:"not null;column:rotate_num"`
// TableName .
func (hp mysqlTS) TableName() string {
	return "tsad_points"
}

type mysqlPoint struct {
	T int64   `json:"T"`
	V float64 `json:"V"`
}

// MysqlMiddleStore .
type MysqlMiddleStore struct {
	db *gorm.DB
}

// NewMysqlMiddleStore .
func NewMysqlMiddleStore(dsn string) (MiddleStorer, error) {
	db, err := gorm.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	return &MysqlMiddleStore{db}, nil
}

// Store .
func (m *MysqlMiddleStore) Store(src Source, ps ts.Points) error {
	key, err := srcKey(src)
	if err != nil {
		return fmt.Errorf("get key from source err: %v, src: %v", err, src)
	}

	mps := make([]mysqlPoint, 0, len(ps))
	for _, p := range ps {
		mps = append(mps, mysqlPoint{
			T: p.Stamp().Unix(),
			V: p.Value(),
		})
	}

	buf, err := json.Marshal(mps)
	if err != nil {
		return fmt.Errorf("json marshal err: %v", err)
	}

	mts := mysqlTS{
		Key:    key,
		Points: string(buf),
	}

	if err := m.db.Save(mts).Error; err != nil {
		return fmt.Errorf("write db err: %v, key: %v", err, key)
	}

	return nil
}

// Fetch .
func (m *MysqlMiddleStore) Fetch(src Source) (ts.Points, error) {
	key, err := srcKey(src)
	if err != nil {
		return nil, fmt.Errorf("get key from source err: %v, src: %v", err, src)
	}

	var mts mysqlTS
	if err := m.db.Where("key=?", key).First(&mts).Error; err != nil {
		return nil, fmt.Errorf("read db err: %v, key: %v", err, key)
	}

	var ps []mysqlPoint
	if err := json.Unmarshal([]byte(mts.Points), &ps); err != nil {
		return nil, fmt.Errorf("invalid point, err: %v, key: %v", err, key)
	}

	tsps := make(ts.Points, 0, len(ps))
	for _, p := range ps {
		tsps = append(tsps, ts.NewPoint(time.Unix(p.T, 0), p.V))
	}

	return tsps, nil
}

func srcKey(src Source) (string, error) {
	// src string may be too long, so do md5 for it
	srcStr := fmt.Sprintf("%v", src)
	h := md5.New()
	_, err := h.Write([]byte(srcStr))
	if err != nil {
		return "", err
	}
	srcMd5 := h.Sum(nil)
	str := base64.StdEncoding.EncodeToString(srcMd5)
	return str, nil
}
