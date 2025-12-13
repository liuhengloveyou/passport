package dao

import (
	"github.com/jackc/pgx/v5"
)

type DAOInterface interface {
	Validate() error
	Insert(tx *pgx.Tx) (int64, error)
	Update(tx *pgx.Tx) (int64, error)
	Delete(tx *pgx.Tx) (int64, error)
	Select(tx *pgx.Tx, pageNo, pageSize uint) (interface{}, error)
}

// func Insert(tx *pgx.Tx, model DAOInterface) (rows int64, e error) {
// 	var _tx *pgx.Tx

// 	if tx != nil {
// 		_tx = tx
// 	} else {
// 		if _tx, e = common.DBPool.Begin(); e != nil {
// 			return -1, e
// 		}
// 	}

// 	rows, e = model.Insert(tx)
// 	if e != nil {
// 		_tx.Rollback()
// 	} else {
// 		_tx.Commit()
// 	}

// 	return
// }

// func Delete(tx *pgx.Tx, model DAOInterface) (rows int64, e error) {
// 	var _tx *pgx.Tx

// 	if tx != nil {
// 		_tx = tx
// 	} else {
// 		if _tx, e = DB.Begin(); e != nil {
// 			return -1, e
// 		}
// 	}

// 	rows, e = model.Delete(tx)
// 	if e != nil {
// 		_tx.Rollback()
// 	} else {
// 		_tx.Commit()
// 	}

// 	return
// }

// func Update(tx *pgx.Tx, model DAOInterface) (rows int64, e error) {
// 	var _tx *pgx.Tx

// 	if tx != nil {
// 		_tx = tx
// 	} else {
// 		if _tx, e = DB.Begin(); e != nil {
// 			return -1, e
// 		}
// 	}

// 	rows, e = model.Update(_tx)
// 	if e != nil {
// 		_tx.Rollback()
// 	} else {
// 		_tx.Commit()
// 	}

// 	return
// }

// func Select(tx *pgx.Tx, pageNo, pageSize uint, model DAOInterface) (rst interface{}, e error) {
// 	var _tx *pgx.Tx

// 	if tx != nil {
// 		_tx = tx
// 	} else {
// 		if _tx, e = DB.Begin(); e != nil {
// 			return -1, e
// 		}
// 	}

// 	rst, e = model.Select(tx, pageNo, pageSize)
// 	if e != nil {
// 		_tx.Rollback()
// 	} else {
// 		_tx.Commit()
// 	}

// 	return
// }
