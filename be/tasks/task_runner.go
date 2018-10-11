package tasks

import (
	"context"
	"fmt"
	"runtime/trace"

	"github.com/go-sql-driver/mysql"

	"pixur.org/pixur/be/status"
)

const (
	maxTaskRetries            = 3
	innoDbDeadlockErrorNumber = 1213
)

type TaskRunner struct {
	run func(context.Context, Task) status.S
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

func runTask(ctx context.Context, task Task) status.S {
	if messy, ok := task.(Messy); ok {
		defer messy.CleanUp()
	}
	var sts status.S
	for i := 0; i < maxTaskRetries; i++ {
		sts = task.Run(ctx)
		if sts == nil {
			return nil
		}
		if cause := sts.Cause(); cause != nil {
			if mysqlErr, ok := cause.(*mysql.MySQLError); ok {
				if mysqlErr.Number == innoDbDeadlockErrorNumber {
					if resettable, ok := task.(Resettable); ok {
						resettable.ResetForRetry()
					}
					continue
				}
			}
		}
		return sts
	}
	return status.InternalErrorf(sts, "Failed to complete task %s after %d tries", task, maxTaskRetries)
}
