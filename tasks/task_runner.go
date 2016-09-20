package tasks

import (
	"pixur.org/pixur/status"

	"github.com/go-sql-driver/mysql"
)

const (
	maxTaskRetries            = 3
	innoDbDeadlockErrorNumber = 1213
)

type TaskRunner struct {
	run func(task Task) status.S
}

func TestTaskRunner(run func(task Task) status.S) *TaskRunner {
	return &TaskRunner{
		run: run,
	}
}

func (r *TaskRunner) Run(task Task) status.S {
	if r != nil && r.run != nil {
		return r.run(task)
	}
	return runTask(task)
}

func runTask(task Task) status.S {
	if messy, ok := task.(Messy); ok {
		defer messy.CleanUp()
	}
	var sts status.S
	for i := 0; i < maxTaskRetries; i++ {
		sts = task.Run()
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
	return status.InternalErrorf(sts, "Failed to complete task %s after %s tries", task, maxTaskRetries)
}
