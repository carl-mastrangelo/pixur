package pixur

import (
	"fmt"
	"testing"

	"github.com/go-sql-driver/mysql"
)

type fakeTask struct {
	runCount   int
	resetCount int
	run        func() error
	reset      func()
}

func (t *fakeTask) Run() error {
	t.runCount++
	if t.run != nil {
		return t.run()
	}
	return nil
}

func (t *fakeTask) Reset() {
	t.resetCount++
	if t.reset != nil {
		t.reset()
	}
}

func TestTaskIsReset_success(t *testing.T) {
	runner := new(TaskRunner)
	task := new(fakeTask)
	err := runner.Run(task)

	if err != nil {
		t.Fatal(err)
	}

	if task.runCount != 1 {
		t.Fatal("Expected task to be run 1 time")
	}

	// We always expect the task to be reset, even in success.
	if task.resetCount != 1 {
		t.Fatal("Expected task to be reset 1 time")
	}
}

func TestTaskIsReset_failure(t *testing.T) {
	runner := new(TaskRunner)
	task := new(fakeTask)
	expectedError := fmt.Errorf("Expected")
	task.run = func() error {
		return expectedError
	}
	err := runner.Run(task)

	if err != expectedError {
		t.Fatal("Expected different error", err)
	}

	if task.runCount != 1 {
		t.Fatal("Expected task to be run 1 time")
	}

	// We always expect the task to be reset, even in success.
	if task.resetCount != 1 {
		t.Fatal("Expected task to be reset 1 time")
	}
}

func TestTaskRetriesOnDeadlock(t *testing.T) {
	runner := new(TaskRunner)
	task := new(fakeTask)

	task.run = func() error {
		return &mysql.MySQLError{Number: innoDbDeadlockErrorNumber}
	}
	err := runner.Run(task)

	if err == nil {
		t.Fatal("Expected error, but was nil")
	}

	if task.runCount != maxTaskRetries {
		t.Fatalf("Expected task to be run %d time", maxTaskRetries)
	}

	// Reset 1 more time than total runs.
	if task.resetCount != maxTaskRetries+1 {
		t.Fatalf("Expected task to be reset %d time", maxTaskRetries)
	}
}

func TestTaskFailsOnOtherError(t *testing.T) {
	runner := new(TaskRunner)
	task := new(fakeTask)

	task.run = func() error {
		return &mysql.MySQLError{Number: innoDbDeadlockErrorNumber + 1}
	}
	err := runner.Run(task)

	if err == nil {
		t.Fatal("Expected error, but was nil")
	}

	if task.runCount != 1 {
		t.Fatal("Expected task to be run 1 time")
	}

	// Reset 1 more time than total runs.
	if task.resetCount != 1 {
		t.Fatal("Expected task to be reset 1 time")
	}
}
