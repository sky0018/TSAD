package manager

import (
	"database/sql"
	"fmt"
	"os"
	"time"
	"code.byted.org/gopkg/env"
)

const (
	defaultDutyTable       = "tsad_duty_lock"
	defaultDutyKeyField    = "lock_key"
	defaultDutyLockedField = "locked_by"
	defaultDutyExpField    = "lock_expiration"

	defaultDutyLockKey = "lock"
)

// DefaultDutyLeaser .
type DefaultDutyLeaser struct {
	distLocker DistLocker
}

// NewDefaultDutyLeaser .
func NewDefaultDutyLeaser(dsn string) (*DefaultDutyLeaser, error) {
	op := &MysqlDistLockOptions{
		Identity:        env.HostIP(),
		DSN:             dsn,
		TableName:       defaultDutyTable,
		KeyField:        defaultDutyKeyField,
		LockedByField:   defaultDutyLockedField,
		ExpirationField: defaultDutyExpField,
	}

	distLocker, err := NewMysqlDistLocker(op)
	if err != nil {
		return nil, err
	}

	createTableAndRecord(dsn)

	return &DefaultDutyLeaser{
		distLocker: distLocker,
	}, nil
}

// Lease .
// default Lease func use the mysql update
func (leaser DefaultDutyLeaser) Lease(lease time.Duration) error {
	return leaser.distLocker.LockLease(defaultDutyLockKey, lease)
}

// Renewal .
func (leaser DefaultDutyLeaser) Renewal(lease time.Duration) error {
	return leaser.distLocker.RenewalLease(defaultDutyLockKey, lease)
}

// Unlease .
func (leaser DefaultDutyLeaser) Unlease() error {
	return leaser.distLocker.Unlock(defaultDutyLockKey)
}

// createTableAndRecord create dutyLockTable and insert the first record if not exist
func createTableAndRecord(dsn string) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open %s err: %v", dsn, err)
	}

	createSQL := "CREATE TABLE IF NOT EXISTS `%v`(" +
		"`%v` VARCHAR(100) NOT NULL ," +
		"`%v` VARCHAR(100) NOT NULL," +
		"`%v` TIMESTAMP NOT NULL," +
		"UNIQUE INDEX uniq_key (`%v`)" +
		")ENGINE=InnoDB DEFAULT CHARSET=utf8;"
	createSQL = fmt.Sprintf(createSQL,
		defaultDutyTable,
		defaultDutyKeyField,
		defaultDutyLockedField,
		defaultDutyExpField,
		defaultDutyKeyField)
	_, err = db.Exec(createSQL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "run %v err: %v", createSQL, err)
	}

	insertSQL := "INSERT INTO `%v` (`%v`, `%v`, `%v`)  VALUES ('%v', '%v', %v)"
	insertSQL = fmt.Sprintf(insertSQL,
		defaultDutyTable,
		defaultDutyKeyField,
		defaultDutyLockedField,
		defaultDutyExpField,
		defaultDutyLockKey,
		env.HostIP(),
		`NOW()`,
	)
	_, err = db.Exec(insertSQL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "run %v err: %v", insertSQL, err)
	}
}

var dutyLeaser *DefaultDutyLeaser

func initDistLock() error {
	locker, err := NewDefaultDutyLeaser(config.MysqlLocklDSN)
	if err != nil {
		return err
	}
	dutyLeaser = locker
	return nil
}

// Lease .
func Lease(lease time.Duration) error {
	return dutyLeaser.Lease(lease)
}

// Renewal .
func Renewal(lease time.Duration) error {
	return dutyLeaser.Renewal(lease)
}
