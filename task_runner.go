package pixur

import (
	"fmt"

	"github.com/go-sql-driver/mysql"
)

const (
	maxTaskRetries            = 3
	innoDbDeadlockErrorNumber = 1213
)

type TaskRunner struct {
}

func (r *TaskRunner) Run(task Task) error {
	defer task.CleanUp()
	for i := 0; i < maxTaskRetries; i++ {
		err := task.Run()
		if err, ok := err.(*mysql.MySQLError); ok {
			if err.Number == innoDbDeadlockErrorNumber {
				//log.Printf("Retrying task %d/%d: %s", i, maxTaskRetries, err)
				// Reset here too, to ensure all state is clean
				task.ResetForRetry()
				continue
			}
			// Something else bad happened.
			return err
		}

		// Not a mysql error
		return err
	}
	return fmt.Errorf("Failed to complete task %s after %s tries", task, maxTaskRetries)
}
