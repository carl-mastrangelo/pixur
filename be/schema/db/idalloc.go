package db

import (
	"context"
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
const DefaultAllocatorGrab = 1

var AllocatorGrab int64 = DefaultAllocatorGrab

type querierExecutor interface {
	Querier
	Executor
}

// Be careful when using this function.  If the transaction fails, alloc will think it has updated
// the cached copy.
func (alloc *IDAlloc) reserve(qe querierExecutor, grab int64, adap DBAdapter) (int64, status.S) {
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
			return 0, status.InternalError(nil, "Too many rows on sequence table")
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
		return 0, status.InternalError(nil, "Too few rows on sequence table")
	}
	buf.Reset()
	buf.WriteString("UPDATE " + adap.Quote(tabname) + " SET " + adap.Quote(colname) + " = ?;")
	if _, err := qe.Exec(buf.String(), num+grab); err != nil {
		return 0, status.From(err)
	}
	return num, nil
}

func (alloc *IDAlloc) refill(ctx context.Context, exec Beginner, grab int64, adap DBAdapter) (stscap status.S) {
	j, err := exec.Begin(ctx)
	if err != nil {
		return status.From(err)
	}
	defer func() {
		if err := j.Rollback(); err != nil {
			status.ReplaceOrSuppress(&stscap, status.From(err))
		}
	}()

	num, sts := alloc.reserve(j, grab, adap)
	if sts != nil {
		return sts
	}

	if err := j.Commit(); err != nil {
		return status.From(err)
	}
	alloc.available += grab
	alloc.next = num
	return nil
}

func AllocID(ctx context.Context, exec Beginner, alloc *IDAlloc, adap DBAdapter) (int64, error) {
	return allocID(ctx, exec, alloc, adap)
}

func allocID(ctx context.Context, exec Beginner, alloc *IDAlloc, adap DBAdapter) (int64, status.S) {
	alloc.lock.Lock()
	defer alloc.lock.Unlock()
	if alloc.available == 0 {
		if sts := alloc.refill(ctx, exec, AllocatorGrab, adap); sts != nil {
			return 0, sts
		}
	}
	num := alloc.next
	alloc.next++
	alloc.available--
	return num, nil
}

func AllocIDJob(qe querierExecutor, alloc *IDAlloc, adap DBAdapter) (int64, error) {
	return alloc.reserve(qe, 1, adap)
}
