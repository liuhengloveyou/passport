package dao

import (
	"database/sql"
	"encoding/gob"

	. "github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"

	_ "github.com/go-sql-driver/mysql"
)

type DAOInterface interface {
	Validate() error
	Insert(tx *sql.Tx) (int64, error)
	Update(tx *sql.Tx) (int64, error)
	Delete(tx *sql.Tx) (int64, error)
	Select(tx *sql.Tx, pageNo, pageSize uint) (interface{}, error)
}

func init() {
	gob.Register(protos.User{})
}

func Insert(tx *sql.Tx, model DAOInterface) (rows int64, e error) {
	var _tx *sql.Tx

	if tx != nil {
		_tx = tx
	} else {
		if _tx, e = DB.Begin(); e != nil {
			return -1, e
		}
	}

	rows, e = model.Insert(tx)
	if e != nil {
		_tx.Rollback()
	} else {
		_tx.Commit()
	}

	return
}

func Delete(tx *sql.Tx, model DAOInterface) (rows int64, e error) {
	var _tx *sql.Tx

	if tx != nil {
		_tx = tx
	} else {
		if _tx, e = DB.Begin(); e != nil {
			return -1, e
		}
	}

	rows, e = model.Delete(tx)
	if e != nil {
		_tx.Rollback()
	} else {
		_tx.Commit()
	}

	return
}

func Update(tx *sql.Tx, model DAOInterface) (rows int64, e error) {
	var _tx *sql.Tx

	if tx != nil {
		_tx = tx
	} else {
		if _tx, e = DB.Begin(); e != nil {
			return -1, e
		}
	}

	rows, e = model.Update(_tx)
	if e != nil {
		_tx.Rollback()
	} else {
		_tx.Commit()
	}

	return
}

func Select(tx *sql.Tx, pageNo, pageSize uint, model DAOInterface) (rst interface{}, e error) {
	var _tx *sql.Tx

	if tx != nil {
		_tx = tx
	} else {
		if _tx, e = DB.Begin(); e != nil {
			return -1, e
		}
	}

	rst, e = model.Select(tx, pageNo, pageSize)
	if e != nil {
		_tx.Rollback()
	} else {
		_tx.Commit()
	}

	return
}
