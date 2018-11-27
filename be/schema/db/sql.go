package db

import (
	"context"
	"database/sql"
	"fmt"
	"runtime/trace"

	"pixur.org/pixur/be/status"
)

type sqlpreprocessor func(string) string

type dbWrapper struct {
	adap  DBAdapter
	db    *sql.DB
	alloc IDAlloc
	// sqlpreprocessor modifies the sql strings just before they are passed to database/sql
	// may be nil
	pp sqlpreprocessor
}

func (w *dbWrapper) Adapter() DBAdapter {
	return w.adap
}

func (w *dbWrapper) IDAllocator() *IDAlloc {
	return &w.alloc
}

var _ Retryable = &sqlError{}
var _ error = &sqlError{}

type sqlError struct {
	wrapped error
	adap    DBAdapter
}

func (e *sqlError) Format(f fmt.State, r rune) {
	if ef, ok := e.wrapped.(fmt.Formatter); ok {
		ef.Format(f, r)
		return
	}
	var err error
	defer func() {
		if err != nil {
			panic(err)
		}
	}()
	switch r {
	case 'v':
		switch {
		case f.Flag('+'):
			_, err = fmt.Fprintf(f, "%+v", e.wrapped)
		case f.Flag('#'):
			_, err = fmt.Fprintf(f, "%#v", e.wrapped)
		default:
			_, err = fmt.Fprintf(f, "%v ==> %#v", e.wrapped, e.wrapped)
		}
	default:
		_, err = fmt.Fprintf(f, "%%!%s(bad fmt for %s)", string(r), e.String())
	}
}

func (e *sqlError) Error() string {
	return e.wrapped.Error()
}

func (e *sqlError) String() string {
	if ws, ok := e.wrapped.(fmt.Stringer); ok {
		return ws.String()
	}
	return e.Error()
}

func (e *sqlError) CanRetry() bool {
	return e.adap.RetryableErr(e.wrapped)
}

func (w *dbWrapper) Begin(ctx context.Context) (QuerierExecutorCommitter, error) {
	return w.begin(ctx)
}

func (w *dbWrapper) begin(ctx context.Context) (*txWrapper, status.S) {
	if trace.IsEnabled() {
		defer trace.StartRegion(ctx, "SqlBeginTx").End()
	}
	tx, err := w.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	if err != nil {
		return nil, status.Unknown(&sqlError{
			wrapped: err,
			adap:    w.adap,
		}, "failed to begin query")
	}
	return &txWrapper{
		tx:   tx,
		ctx:  ctx,
		adap: w.adap,
		pp:   w.pp,
	}, nil
}

func (w *dbWrapper) Close() error {
	return w._close()
}

func (w *dbWrapper) _close() status.S {
	if err := w.db.Close(); err != nil {
		return status.Unknown(&sqlError{
			wrapped: err,
			adap:    w.adap,
		}, "can't close db")
	}
	return nil
}

func (w *dbWrapper) InitSchema(ctx context.Context, tables []string) error {
	return w.initSchema(ctx, tables)
}

func (w *dbWrapper) initSchema(ctx context.Context, tables []string) status.S {
	if trace.IsEnabled() {
		defer trace.StartRegion(ctx, "SqlInitSchema").End()
	}

	// also includes initial data
	for _, table := range tables {
		if _, err := w.db.ExecContext(ctx, table); err != nil {
			return status.Unknown(&sqlError{
				wrapped: err,
				adap:    w.adap,
			}, "can't init table", table)
		}
	}
	return nil
}

type txWrapper struct {
	ctx  context.Context
	tx   *sql.Tx
	pp   sqlpreprocessor
	adap DBAdapter
	done bool
}

func (w *txWrapper) Exec(query string, args ...interface{}) (Result, error) {
	return w.exec(query, args...)
}

func (w *txWrapper) exec(query string, args ...interface{}) (Result, status.S) {
	if trace.IsEnabled() {
		defer trace.StartRegion(w.ctx, "SqlExec").End()
	}
	if w.pp != nil {
		query = w.pp(query)
	}
	res, err := w.tx.ExecContext(w.ctx, query, args...)
	if err != nil {
		return nil, status.Unknown(&sqlError{
			wrapped: err,
			adap:    w.adap,
		}, "can't exec")
	}
	return Result(res), nil
}

func (w *txWrapper) Query(query string, args ...interface{}) (Rows, error) {
	return w._query(query, args...)
}

func (w *txWrapper) _query(query string, args ...interface{}) (Rows, status.S) {
	if trace.IsEnabled() {
		defer trace.StartRegion(w.ctx, "SqlQuery").End()
	}
	if w.pp != nil {
		query = w.pp(query)
	}
	rows, err := w.tx.QueryContext(w.ctx, query, args...)
	if err != nil {
		return nil, status.Unknown(&sqlError{
			wrapped: err,
			adap:    w.adap,
		}, "can't query")
	}
	return Rows(rows), nil
}

func (w *txWrapper) Commit() error {
	return w.commit()
}

func (w *txWrapper) commit() status.S {
	if trace.IsEnabled() {
		defer trace.StartRegion(w.ctx, "SqlCommit").End()
	}
	if err := w.tx.Commit(); err != nil {
		return status.Unknown(&sqlError{
			wrapped: err,
			adap:    w.adap,
		}, "can't commit")
	}
	w.done = true
	return nil
}

func (w *txWrapper) Rollback() error {
	return w.rollback()
}

func (w *txWrapper) rollback() status.S {
	if w.done {
		return nil
	}
	w.done = true
	if trace.IsEnabled() {
		defer trace.StartRegion(w.ctx, "SqlRollback").End()
	}
	if err := w.tx.Rollback(); err != nil {
		return status.Unknown(&sqlError{
			wrapped: err,
			adap:    w.adap,
		}, "can't rollback")
	}
	return nil
}
