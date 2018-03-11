package tasks

import (
	"testing"

	"pixur.org/pixur/be/status"

	"github.com/go-sql-driver/mysql"
)

type fakeTask struct {
	runCount     int
	resetCount   int
	cleanUpCount int
	run          func() status.S
	reset        func()
	cleanUp      func()
}

func (t *fakeTask) Run() status.S {
	t.runCount++
	if t.run != nil {
		return t.run()
	}
	return nil
}

func (t *fakeTask) ResetForRetry() {
	t.resetCount++
	if t.reset != nil {
		t.reset()
	}
}

func (t *fakeTask) CleanUp() {
	t.cleanUpCount++
	if t.cleanUp != nil {
		t.cleanUp()
	}
}

func TestTaskRunnerNil(t *testing.T) {
	var tr *TaskRunner
	sts := tr.Run(new(fakeTask))
	if sts != nil {
		t.Error("error running task", sts)
	}
}

func TestTaskIsNotReset_success(t *testing.T) {
	runner := new(TaskRunner)
	task := new(fakeTask)
	sts := runner.Run(task)

	if sts != nil {
		t.Fatal(sts)
	}

	if task.runCount != 1 {
		t.Fatal("Expected task to be run 1 time")
	}

	if task.resetCount != 0 {
		t.Fatal("Expected task to be reset 0 times")
	}
	if task.cleanUpCount != 1 {
		t.Fatal("Expected task to be cleaned up 1 times")
	}
}

func TestTaskIsReset_failure(t *testing.T) {
	runner := new(TaskRunner)
	task := new(fakeTask)
	expectedError := status.InternalError(nil, "Expected")
	task.run = func() status.S {
		return expectedError
	}
	sts := runner.Run(task)

	if sts != expectedError {
		t.Fatal("Expected different error", sts)
	}

	if task.runCount != 1 {
		t.Fatal("Expected task to be run 1 time")
	}

	if task.resetCount != 0 {
		t.Fatal("Expected task to be reset 0 times")
	}
	if task.cleanUpCount != 1 {
		t.Fatal("Expected task to be cleaned up 1 times")
	}
}

func TestTaskRetriesOnDeadlock(t *testing.T) {
	runner := new(TaskRunner)
	task := new(fakeTask)

	task.run = func() status.S {
		return status.InternalError(&mysql.MySQLError{Number: innoDbDeadlockErrorNumber}, "")
	}
	sts := runner.Run(task)

	if sts == nil {
		t.Fatal("Expected error, but was nil")
	}

	if task.runCount != maxTaskRetries {
		t.Fatalf("Expected task to be run %d time", maxTaskRetries)
	}

	if task.resetCount != maxTaskRetries {
		t.Fatalf("Expected task to be reset %d time", maxTaskRetries)
	}
	if task.cleanUpCount != 1 {
		t.Fatal("Expected task to be cleaned up 1 times")
	}
}

func TestTaskFailsOnOtherError(t *testing.T) {
	runner := new(TaskRunner)
	task := new(fakeTask)

	task.run = func() status.S {
		return status.InternalError(&mysql.MySQLError{Number: innoDbDeadlockErrorNumber + 1}, "")
	}
	sts := runner.Run(task)

	if sts == nil {
		t.Fatal("Expected error, but was nil")
	}

	if task.runCount != 1 {
		t.Fatal("Expected task to be run 1 time")
	}

	if task.resetCount != 0 {
		t.Fatal("Expected task to be reset 0 times")
	}
	if task.cleanUpCount != 1 {
		t.Fatal("Expected task to be cleaned up 1 times")
	}
}
