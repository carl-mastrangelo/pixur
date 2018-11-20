package db

import (
	"context"
	"math"
	"runtime/trace"
	"strings"
	"sync"

	"pixur.org/pixur/be/status"
)

type IDAlloc struct {
	available int64
	next      int64
	lock      sync.Mutex
}

const (
	SequenceTableName = "_SequenceTable"
	SequenceColName   = "the_sequence"
)

// DefaultAllocatorGrab determines how many IDs will be grabbed at a time. If the number is too high
// program restarts will waste ID space.  Additionally, IDs will become less monotonic if there are
// multiple servers.  If the number is too low, it will make a lot more queries than necessary.
const DefaultAllocatorGrab = 10

var AllocatorGrab int64 = DefaultAllocatorGrab

type querierExecutor interface {
	Querier
	Executor
}

func reserveIDLocked(qe querierExecutor, grab int64, adap DBAdapter) (int64, status.S) {
	if grab < 1 {
		return 0, status.Internal(nil, "grab too low", grab)
	}
	tabname, colname := SequenceTableName, SequenceColName
	var buf strings.Builder
	buf.WriteString("SELECT " + adap.Quote(colname) + " FROM " + adap.Quote(tabname))
	adap.LockStmt(&buf, LockWrite)
	buf.WriteRune(';')

	var num int64
	rows, err := qe.Query(buf.String())
	if err != nil {
		return 0, status.From(err)
	}
	done := false
	for rows.Next() {
		if done {
			return 0, status.Internal(nil, "Too many rows on sequence table")
		}
		if err := rows.Scan(&num); err != nil {
			return 0, status.From(err)
		}
		done = true
	}
	if err := rows.Err(); err != nil {
		return 0, status.From(err)
	}
	if !done {
		return 0, status.Internal(nil, "Too few rows on sequence table")
	}
	if grab > math.MaxInt64-num {
		return 0, status.Internalf(nil, "id allocation overflow %d+%d", num, grab)
	}
	buf.Reset()
	buf.WriteString("UPDATE " + adap.Quote(tabname) + " SET " + adap.Quote(colname) + " = ?;")
	if _, err := qe.Exec(buf.String(), num+grab); err != nil {
		return 0, status.From(err)
	}
	return num, nil
}

func AllocID(ctx context.Context, exec Beginner, alloc *IDAlloc, adap DBAdapter) (int64, error) {
	return allocID(ctx, exec, alloc, adap)
}

func allocID(ctx context.Context, exec Beginner, alloc *IDAlloc, adap DBAdapter) (
	_ int64, stscap status.S) {
	if trace.IsEnabled() {
		defer trace.StartRegion(ctx, "AllocID").End()
	}
	var id int64
	alloc.lock.Lock()
	if alloc.available > 0 {
		id, alloc.next, alloc.available = alloc.next, alloc.next+1, alloc.available-1
		alloc.lock.Unlock()
		return id, nil
	}
	alloc.lock.Unlock()
	// The transaction has to begin outside of the lock to avoid a deadlock.  The lock ordering
	// must be database connection, then idalloc lock.
	j, err := exec.Begin(ctx)
	if err != nil {
		return 0, status.From(err)
	}
	defer func() {
		if err := j.Rollback(); err != nil {
			status.ReplaceOrSuppress(&stscap, status.From(err))
		}
	}()

	grab := AllocatorGrab
	alloc.lock.Lock()
	defer alloc.lock.Unlock()
	// Another thread could have jumped in line and updated allocate.  Double check there isn't
	// actually one available.
	if alloc.available > 0 {
		id, alloc.next, alloc.available = alloc.next, alloc.next+1, alloc.available-1
		return id, nil
	}
	next, sts := reserveIDLocked(j, grab, adap)
	if sts != nil {
		return 0, sts
	}
	if err := j.Commit(); err != nil {
		return 0, status.From(err)
	}
	id, alloc.next, alloc.available = next, next+1, alloc.available+grab-1
	return id, nil
}

func AllocIDJob(ctx context.Context, qe querierExecutor, alloc *IDAlloc, adap DBAdapter) (
	int64, error) {
	if trace.IsEnabled() {
		defer trace.StartRegion(ctx, "AllocIDJob").End()
	}
	alloc.lock.Lock()
	defer alloc.lock.Unlock()
	// Since the transaction may not be commited, don't update alloc
	var id int64
	if alloc.available > 0 {
		id, alloc.next, alloc.available = alloc.next, alloc.next+1, alloc.available-1
		return id, nil
	}
	return reserveIDLocked(qe /*grab=*/, 1, adap)
}
