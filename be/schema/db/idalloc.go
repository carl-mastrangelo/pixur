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

// IdAlloc is an id allocator
type IdAlloc struct {
	idrs         []*idRange
	total        int64
	lowat, hiwat *int64
	lock         sync.Mutex
}

// GetWatermark returns the low and high watermark.  See SetWatermark.
func (alloc *IdAlloc) GetWatermark() (lo, hi int64) {
	alloc.lock.Lock()
	defer alloc.lock.Unlock()

	return alloc.getWatermarkLocked()
}

func (alloc *IdAlloc) getWatermarkLocked() (lo, hi int64) {
	if alloc.lowat == nil || alloc.hiwat == nil {
		return DefaultIdLowWaterMark, DefaultIdHighWaterMark
	}
	return *alloc.lowat, *alloc.hiwat
}

// SetWatermark sets the low and high watermark for Id allocation.  See PreallocateIds.
func (alloc *IdAlloc) SetWatermark(newlo, newhi int64) {
	if newlo > newhi {
		panic("bad watermarks")
	}
	alloc.lock.Lock()
	defer alloc.lock.Unlock()
	alloc.lowat, alloc.hiwat = &newlo, &newhi
}

const (
	SequenceTableName = "_SequenceTable"
	SequenceColName   = "the_sequence"
)

// Watermarks determines how many Ids will be grabbed at a time. If the number is too high
// program restarts will waste Id space.  Additionally, Ids will become less monotonic if there are
// multiple servers.  If the number is too low, it will make a lot more queries than necessary.
const (
	DefaultIdLowWaterMark  = 1
	DefaultIdHighWaterMark = 10
)

type querierExecutor interface {
	Querier
	Executor
}

func reserveIds(qe querierExecutor, grab int64, adap DBAdapter) (int64, status.S) {
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

// PreallocateIds attempts to fill the cache in IdAlloc.  It is best effort.  If the number
// of cached Ids goes below the low watermark, PreallocateIds will attempt to get more. It will
// attempt to refill up to the high watermark.
func PreallocateIds(ctx context.Context, beg Beginner, alloc *IdAlloc, adap DBAdapter) error {
	return preallocateIds(ctx, beg, alloc, adap)
}

func preallocateIds(ctx context.Context, beg Beginner, alloc *IdAlloc, adap DBAdapter) (
	stscap status.S) {
	if trace.IsEnabled() {
		defer trace.StartRegion(ctx, "PreallocateIds").End()
	}
	alloc.lock.Lock()
	lowat, hiwat := alloc.getWatermarkLocked()
	available := alloc.total
	if available >= lowat {
		alloc.lock.Unlock()
		return nil
	}
	alloc.lock.Unlock()
	qec, err := beg.Begin(ctx, false)
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
	lowat, hiwat = alloc.getWatermarkLocked()
	available = alloc.total
	if available >= lowat {
		return nil
	}
	// hiwat >= lowat > available
	grab := hiwat - available
	next, sts := reserveIds(qec, grab, adap)
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
func AllocId(ctx context.Context, beg Beginner, alloc *IdAlloc, adap DBAdapter) (int64, error) {
	return allocId(ctx, beg, alloc, adap)
}

func allocId(ctx context.Context, beg Beginner, alloc *IdAlloc, adap DBAdapter) (
	_ int64, stscap status.S) {
	if trace.IsEnabled() {
		defer trace.StartRegion(ctx, "AllocId").End()
	}
	if sts := preallocateIds(ctx, beg, alloc, adap); sts != nil {
		return 0, sts
	}
	qec, err := beg.Begin(ctx, false)
	if err != nil {
		return 0, status.From(err)
	}
	defer func() {
		if err := qec.Rollback(); err != nil {
			status.ReplaceOrSuppress(&stscap, status.From(err))
		}
	}()

	id, sts := allocIdJob(ctx, qec, alloc, adap)
	if sts != nil {
		return 0, sts
	}
	if err := qec.Commit(); err != nil {
		return 0, status.From(err)
	}
	return id, nil
}

func AllocIdJob(ctx context.Context, qe querierExecutor, alloc *IdAlloc, adap DBAdapter) (
	int64, error) {
	return allocIdJob(ctx, qe, alloc, adap)
}

func allocIdJob(ctx context.Context, qe querierExecutor, alloc *IdAlloc, adap DBAdapter) (
	int64, status.S) {
	if trace.IsEnabled() {
		defer trace.StartRegion(ctx, "AllocIdJob").End()
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
	return reserveIds(qe /*grab=*/, 1, adap)
}
