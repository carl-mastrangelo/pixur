package pixur

import (
	"fmt"
	"testing"

	"github.com/go-sql-driver/mysql"
)

type fakeTask struct {
	runCount     int
	resetCount   int
	cleanUpCount int
	run          func() error
	reset        func()
	cleanUp      func()
}

func (t *fakeTask) Run() error {
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

func TestTaskIsNotReset_success(t *testing.T) {
	runner := new(TaskRunner)
	task := new(fakeTask)
	err := runner.Run(task)

	if err != nil {
		t.Fatal(err)
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

	if task.resetCount != 0 {
		t.Fatal("Expected task to be reset 0 times")
	}
	if task.cleanUpCount != 1 {
		t.Fatal("Expected task to be cleaned up 1 times")
	}
}
