package db

import (
	"context"
	"math"
	"runtime/trace"
	"strings"
	"sync"

	"pixur.org/pixur/be/status"
)

type idRange struct {
	next, available int64
}

type IDAlloc struct {
	idrs  []*idRange
	total int64
	lock  sync.Mutex
}

const (
	SequenceTableName = "_SequenceTable"
	SequenceColName   = "the_sequence"
)

// DefaultAllocatorGrab determines how many IDs will be grabbed at a time. If the number is too high
// program restarts will waste ID space.  Additionally, IDs will become less monotonic if there are
// multiple servers.  If the number is too low, it will make a lot more queries than necessary.
const (
	DefaultIDLowWaterMark  = 1
	DefaultIDHighWaterMark = 10
)

var IDLowWaterMark int64 = DefaultIDLowWaterMark
var IDHighWaterMark int64 = DefaultIDHighWaterMark

type querierExecutor interface {
	Querier
	Executor
}

func reserveIDs(qe querierExecutor, grab int64, adap DBAdapter) (int64, status.S) {
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

// PreallocateIDs attempts to fill the cache in IDAlloc.   It is best effort.
func PreallocateIDs(ctx context.Context, beg Beginner, alloc *IDAlloc, adap DBAdapter) error {
	return preallocateIDs(ctx, beg, alloc, adap)
}

func preallocateIDs(ctx context.Context, beg Beginner, alloc *IDAlloc, adap DBAdapter) (
	stscap status.S) {
	if trace.IsEnabled() {
		defer trace.StartRegion(ctx, "PreallocateIDs").End()
	}
	alloc.lock.Lock()
	if alloc.total >= IDLowWaterMark {
		alloc.lock.Unlock()
		return nil
	}
	alloc.lock.Unlock()
	qec, err := beg.Begin(ctx)
	if err != nil {
		return status.From(err)
	}
	defer func() {
		if err := qec.Rollback(); err != nil {
			status.ReplaceOrSuppress(&stscap, status.From(err))
		}
	}()
	alloc.lock.Lock()
	defer alloc.lock.Unlock()
	available := alloc.total
	if available >= IDLowWaterMark {
		return nil
	}

	grab := IDHighWaterMark - available
	next, sts := reserveIDs(qec, grab, adap)
	if sts != nil {
		return sts
	}
	if err := qec.Commit(); err != nil {
		return status.From(err)
	}
	alloc.idrs = append(alloc.idrs, &idRange{next: next, available: grab})
	alloc.total += grab
	return nil
}

// Don't use this unless outside of an existing Job.
func AllocID(ctx context.Context, beg Beginner, alloc *IDAlloc, adap DBAdapter) (int64, error) {
	return allocID(ctx, beg, alloc, adap)
}

func allocID(ctx context.Context, beg Beginner, alloc *IDAlloc, adap DBAdapter) (
	_ int64, stscap status.S) {
	if trace.IsEnabled() {
		defer trace.StartRegion(ctx, "AllocID").End()
	}
	if sts := preallocateIDs(ctx, beg, alloc, adap); sts != nil {
		return 0, sts
	}
	qec, err := beg.Begin(ctx)
	if err != nil {
		return 0, status.From(err)
	}
	defer func() {
		if err := qec.Rollback(); err != nil {
			status.ReplaceOrSuppress(&stscap, status.From(err))
		}
	}()

	id, sts := allocIDJob(ctx, qec, alloc, adap)
	if sts != nil {
		return 0, sts
	}
	if err := qec.Commit(); err != nil {
		return 0, status.From(err)
	}
	return id, nil
}

func AllocIDJob(ctx context.Context, qe querierExecutor, alloc *IDAlloc, adap DBAdapter) (
	int64, error) {
	return allocIDJob(ctx, qe, alloc, adap)
}

func allocIDJob(ctx context.Context, qe querierExecutor, alloc *IDAlloc, adap DBAdapter) (
	int64, status.S) {
	if trace.IsEnabled() {
		defer trace.StartRegion(ctx, "AllocIDJob").End()
	}
	alloc.lock.Lock()
	if alloc.total > 0 {
		for i, idr := range alloc.idrs {
			if idr.available > 0 {
				var id int64
				id, idr.next, idr.available, alloc.total =
					idr.next, idr.next+1, idr.available-1, alloc.total-1
				alloc.idrs = alloc.idrs[i:]
				alloc.lock.Unlock()
				return id, nil
			}
		}
		panic("unreachable")
	}
	alloc.lock.Unlock()
	// Since the transaction may not be commited, don't update alloc
	return reserveIDs(qe /*grab=*/, 1, adap)
}
