package tasks

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"runtime/trace"
	"time"

	"pixur.org/pixur/be/schema/db"
	"pixur.org/pixur/be/status"
)

const (
	maxTaskRetries = 5
)

var thetasklogger *log.Logger = log.New(os.Stderr, "", log.LstdFlags)

type TaskRunner struct {
	run    func(context.Context, Task) status.S
	thelog *log.Logger
}

func TestTaskRunner(run func(context.Context, Task) status.S) *TaskRunner {
	return &TaskRunner{
		run: run,
	}
}

func (r *TaskRunner) Run(ctx context.Context, task Task) status.S {
	if trace.IsEnabled() {
		var tracetask *trace.Task
		ctx, tracetask = trace.NewTask(ctx, fmt.Sprintf("%T", task))
		defer tracetask.End()
	}
	if r != nil && r.run != nil {
		return r.run(ctx, task)
	}
	return runTask(ctx, task)
}

func stsSliceToErrSlice(ss []status.S) []error {
	es := make([]error, 0, len(ss))
	for _, s := range ss {
		es = append(es, s)
	}
	return es
}

var (
	taskInitialBackoffDelay      = time.Second
	taskInitialBackoffMultiplier = math.Phi
)

func runTask(ctx context.Context, task Task) (stscap status.S) {
	var failures []status.S
	defer func() {
		if stscap == nil && len(failures) != 0 {
			if thetasklogger != nil {
				warnsts := status.Unknown(nil, "Encountered errors while running")
				warnsts = status.WithSuppressed(warnsts, stsSliceToErrSlice(failures)...)

				thetasklogger.Println(warnsts)
			}
		}
	}()

	backoff := taskInitialBackoffDelay
	deadline, deadlineok := ctx.Deadline()
	for i := 0; i < maxTaskRetries; i++ {
		select {
		case <-ctx.Done():
			failures = append(failures, status.From(ctx.Err()))
			break
		default:
		}
		sts := task.Run(ctx)
		if sts == nil {
			return nil
		}
		failures = append(failures, sts)

		if cause := unwrapTaskStatus(sts); cause != nil {
			if retryable, ok := cause.(db.Retryable); ok {
				if retryable.CanRetry() {
					sleep := backoff
					backoff = time.Duration(float64(backoff) * taskInitialBackoffMultiplier)
					wakeup := time.Now().Add(sleep)
					if deadlineok && wakeup.After(deadline) {
						sts := status.DeadlineExceededf(
							nil, "not enough time to complete %T (%v < %v)", task, deadline, wakeup)
						return status.WithSuppressed(sts, stsSliceToErrSlice(failures)...)
					}
					select {
					case <-time.After(sleep):
					case <-ctx.Done():
						return status.WithSuppressed(status.From(ctx.Err()), stsSliceToErrSlice(failures)...)
					}
					continue
				}
			}
		}
		return status.WithSuppressed(sts, stsSliceToErrSlice(failures[:len(failures)-1])...)
	}
	if thetasklogger != nil {
		thetasklogger.Printf("Failed to complete task %T after %d tries", task, maxTaskRetries)
	}
	return status.WithSuppressed(
		failures[len(failures)-1], stsSliceToErrSlice(failures[:len(failures)-1])...)
}

func unwrapTaskStatus(sts status.S) error {
	var cause error = sts
	for {
		if s, ok := cause.(status.S); ok {
			cause = s.Cause()
		} else {
			break
		}
	}
	return cause
}
