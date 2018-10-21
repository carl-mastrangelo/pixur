package tasks

import (
	"context"
	"testing"

	"pixur.org/pixur/be/status"
)

type fakeTask struct {
	runCount   int
	resetCount int
	run        func(context.Context) status.S
	reset      func()
}

func (t *fakeTask) Run(ctx context.Context) status.S {
	t.runCount++
	if t.run != nil {
		return t.run(ctx)
	}
	return nil
}

func (t *fakeTask) ResetForRetry() {
	t.resetCount++
	if t.reset != nil {
		t.reset()
	}
}

func TestTaskRunnerNil(t *testing.T) {
	var tr *TaskRunner
	sts := tr.Run(context.Background(), new(fakeTask))
	if sts != nil {
		t.Error("error running task", sts)
	}
}

func TestTaskIsNotReset_success(t *testing.T) {
	runner := new(TaskRunner)
	task := new(fakeTask)
	sts := runner.Run(context.Background(), task)

	if sts != nil {
		t.Fatal(sts)
	}

	if task.runCount != 1 {
		t.Fatal("Expected task to be run 1 time")
	}

	if task.resetCount != 0 {
		t.Fatal("Expected task to be reset 0 times")
	}
}

func TestTaskIsReset_failure(t *testing.T) {
	runner := new(TaskRunner)
	task := new(fakeTask)
	expectedError := status.InternalError(nil, "Expected")
	task.run = func(_ context.Context) status.S {
		return expectedError
	}
	sts := runner.Run(context.Background(), task)

	if sts != expectedError {
		t.Fatal("Expected different error", sts)
	}

	if task.runCount != 1 {
		t.Fatal("Expected task to be run 1 time")
	}

	if task.resetCount != 0 {
		t.Fatal("Expected task to be reset 0 times")
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
	sts := runner.Run(context.Background(), task)
	thetasklogger = oldlog

	if sts == nil {
		t.Fatal("Expected error, but was nil")
	}

	if task.runCount != maxTaskRetries {
		t.Fatalf("Expected task to be run %d times !%d", maxTaskRetries, task.runCount)
	}
	if task.resetCount != maxTaskRetries {
		t.Fatalf("Expected task to be reset %d times, !%d", maxTaskRetries, task.resetCount)
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

	if task.resetCount != 0 {
		t.Fatal("Expected task to be reset 0 times")
	}
}
