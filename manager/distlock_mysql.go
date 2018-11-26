package manager

import (
	"database/sql"
	"fmt"
	"time"
	"code.byted.org/gopkg/env"
)

// DistLocker distributed lock
type DistLocker interface {
	LockLease(key string, lease time.Duration) error
	RenewalLease(key string, lease time.Duration) error
	Unlock(key string) error
}

// MysqlDistLockOptions .
type MysqlDistLockOptions struct {
	Identity        string
	DSN             string
	TableName       string
	KeyField        string
	LockedByField   string
	ExpirationField string
}

// MysqlDistLocker implement DistLocker based on mysql
type MysqlDistLocker struct {
	leaseSQL   string
	renewalSQL string
	unlockSQL  string

	db *sql.DB
	op *MysqlDistLockOptions
}

// NewMysqlDistLocker .
func NewMysqlDistLocker(op *MysqlDistLockOptions) (*MysqlDistLocker, error) {
	if op.Identity == "" {
		op.Identity = env.HostIP()
	}
	if op.DSN == "" {
		return nil, fmt.Errorf("no DSN")
	}
	if op.TableName == "" {
		return nil, fmt.Errorf("no tablename")
	}
	if op.KeyField == "" {
		return nil, fmt.Errorf("no key field")
	}
	if op.LockedByField == "" {
		return nil, fmt.Errorf("no locked by field")
	}
	if op.ExpirationField == "" {
		return nil, fmt.Errorf("no expiration field")
	}

	db, err := sql.Open("mysql", op.DSN)
	if err != nil {
		return nil, err
	}

	leaseSQL := fmt.Sprintf("update `%v` set `%v`=?, `%v`=? where `%v`=? and `%v`<?",
		op.TableName, op.LockedByField, op.ExpirationField, op.KeyField, op.ExpirationField)
	renewalSQL := fmt.Sprintf("update `%v` set `%v`=? where `%v`=? and `%v`=? and `%v`>?",
		op.TableName, op.ExpirationField, op.KeyField, op.LockedByField, op.ExpirationField)
	unlockSQL := fmt.Sprintf("update `%v` set `%v`=? where `%v`=? and `%v`=?",
		op.TableName, op.ExpirationField, op.KeyField, op.LockedByField)

	return &MysqlDistLocker{
		leaseSQL:   leaseSQL,
		renewalSQL: renewalSQL,
		unlockSQL:  unlockSQL,
		db:         db,
		op:         op,
	}, nil
}

// LockLease .
func (mdl *MysqlDistLocker) LockLease(key string, lease time.Duration) error {
	// update table set lockedBy=this.ID, expiration=newExp() where key=this.key and expiration < now()
	result, err := mdl.db.Exec(mdl.leaseSQL,
		mdl.op.Identity,
		time.Now().Add(lease),
		key,
		time.Now())
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("key not found: %v", key)
	}
	return nil
}

// RenewalLease .
func (mdl *MysqlDistLocker) RenewalLease(key string, lease time.Duration) error {
	// update table set expiration=now()+lease where key=this.key and lockedBy=this.ID and expiration>now()
	now := time.Now()
	newExp := now.Add(lease)
	result, err := mdl.db.Exec(mdl.renewalSQL,
		newExp,
		key,
		mdl.op.Identity,
		now)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("key not found: %v", key)
	}
	return nil
}

// Unlock .
func (mdl *MysqlDistLocker) Unlock(key string) error {
	// update table set expiration=zero() where key=this.key and lockedBy=this.ID
	_, err := mdl.db.Exec(mdl.unlockSQL,
		&time.Time{},
		key,
		mdl.op.Identity)
	if err != nil {
		return err
	}
	return nil
}
