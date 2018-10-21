package tasks // import "pixur.org/pixur/be/tasks"

import (
	"context"

	"pixur.org/pixur/be/schema/db"
	"pixur.org/pixur/be/status"
)

type Task interface {
	Run(context.Context) status.S
}

// Tasks implement the Resettable interface if they want to run any sort of reset logic.
// This includes things like clearing intermediate results.
type Resettable interface {
	/* If there was a retriable error, this will be called before Run */
	ResetForRetry()
}

func revert(j db.Commiter, stscap *status.S) {
	if err := j.Rollback(); err != nil {
		status.ReplaceOrSuppress(stscap, status.From(err))
	}
}
