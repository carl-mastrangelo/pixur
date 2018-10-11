package db

import (
	"context"
	"database/sql"
	"runtime/trace"
)

type sqlpreprocessor func(string) string

type dbWrapper struct {
	adap DBAdapter
	db   *sql.DB
	// sqlpreprocessor modifies the sql strings just before they are passed to database/sql
	// may be nil
	pp sqlpreprocessor
}

func (w dbWrapper) Adapter() DBAdapter {
	return w.adap
}

func (w dbWrapper) Begin(ctx context.Context) (QuerierExecutorCommitter, error) {
	if trace.IsEnabled() {
		defer trace.StartRegion(ctx, "SqlBeginTx").End()
	}
	tx, err := w.db.Begin()
	return txWrapper{
		tx:  tx,
		ctx: ctx,
		pp:  w.pp,
	}, err
}

func (w dbWrapper) Close() error {
	return w.db.Close()
}

func (w dbWrapper) InitSchema(ctx context.Context, tables []string) error {
	if trace.IsEnabled() {
		defer trace.StartRegion(ctx, "SqlInitSchema").End()
	}
	// also includes initial data
	for _, table := range tables {
		if _, err := w.db.Exec(table); err != nil {
			return err
		}
	}

	return nil
}

type txWrapper struct {
	ctx context.Context
	tx  *sql.Tx
	pp  sqlpreprocessor
}

func (w txWrapper) Exec(query string, args ...interface{}) (Result, error) {
	if trace.IsEnabled() {
		defer trace.StartRegion(w.ctx, "SqlExec").End()
	}
	if w.pp != nil {
		query = w.pp(query)
	}
	res, err := w.tx.ExecContext(w.ctx, query, args...)
	return Result(res), err
}

func (w txWrapper) Query(query string, args ...interface{}) (Rows, error) {
	if trace.IsEnabled() {
		defer trace.StartRegion(w.ctx, "SqlQuery").End()
	}
	if w.pp != nil {
		query = w.pp(query)
	}
	rows, err := w.tx.QueryContext(w.ctx, query, args...)
	return Rows(rows), err
}

func (w txWrapper) Commit() error {
	if trace.IsEnabled() {
		defer trace.StartRegion(w.ctx, "SqlCommit").End()
	}
	return w.tx.Commit()
}

func (w txWrapper) Rollback() error {
	if trace.IsEnabled() {
		defer trace.StartRegion(w.ctx, "SqlRollback").End()
	}
	return w.tx.Rollback()
}
