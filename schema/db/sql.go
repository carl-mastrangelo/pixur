package db

import (
	"database/sql"
)

type dbWrapper struct {
	adap DBAdapter
	db   *sql.DB
}

func (w dbWrapper) Adapter() DBAdapter {
	return w.adap
}

func (w dbWrapper) Begin() (QuerierExecutorCommitter, error) {
	tx, err := w.db.Begin()
	return txWrapper{tx}, err
}

func (w dbWrapper) Close() error {
	return w.db.Close()
}

func (w dbWrapper) InitSchema(tables []string) error {
	// also includes initial data
	for _, table := range tables {
		if _, err := w.db.Exec(table); err != nil {
			return err
		}
	}

	return nil
}

type txWrapper struct {
	tx *sql.Tx
}

func (w txWrapper) Exec(query string, args ...interface{}) (Result, error) {
	res, err := w.tx.Exec(query, args...)
	return Result(res), err
}

func (w txWrapper) Query(query string, args ...interface{}) (Rows, error) {
	rows, err := w.tx.Query(query, args...)
	return Rows(rows), err
}

func (w txWrapper) Commit() error {
	return w.tx.Commit()
}

func (w txWrapper) Rollback() error {
	return w.tx.Rollback()
}
