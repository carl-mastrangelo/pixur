package tasks

import (
	"context"
	"testing"

	"pixur.org/pixur/be/status"
)

type fakeTask struct {
	runCount int
	run      func(context.Context) status.S
}

func (t *fakeTask) Run(ctx context.Context) status.S {
	t.runCount++
	if t.run != nil {
		return t.run(ctx)
	}
	return nil
}

func TestTaskRunnerNil(t *testing.T) {
	var tr *TaskRunner
	sts := tr.Run(context.Background(), new(fakeTask))
	if sts != nil {
		t.Error("error running task", sts)
	}
}

type fakeError bool

func (e *fakeError) Error() string {
	return "bad"
}

func (e *fakeError) CanRetry() bool {
	return bool(*e)
}

func TestTaskRetriesOnRetryableError(t *testing.T) {
	runner := new(TaskRunner)
	task := new(fakeTask)

	task.run = func(_ context.Context) status.S {
		err := fakeError(true)
		return status.InternalError(&err, "err")
	}
	oldlog := thetasklogger
	thetasklogger = nil
	oldtaskInitialBackoffDelay := taskInitialBackoffDelay
	taskInitialBackoffDelay = 1
	sts := runner.Run(context.Background(), task)
	taskInitialBackoffDelay = oldtaskInitialBackoffDelay
	thetasklogger = oldlog

	if sts == nil {
		t.Fatal("Expected error, but was nil")
	}

	if task.runCount != maxTaskRetries {
		t.Fatalf("Expected task to be run %d times !%d", maxTaskRetries, task.runCount)
	}
}

func TestTaskFailsOnOtherError(t *testing.T) {
	runner := new(TaskRunner)
	task := new(fakeTask)

	task.run = func(_ context.Context) status.S {
		err := fakeError(false)
		return status.InternalError(&err, "")
	}
	sts := runner.Run(context.Background(), task)

	if sts == nil {
		t.Fatal("Expected error, but was nil")
	}

	if task.runCount != 1 {
		t.Fatal("Expected task to be run 1 time")
	}
}
