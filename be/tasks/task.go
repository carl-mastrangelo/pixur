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

// Tasks implement the Messy interface if they have side effects outside of the normal
// database transactions.  This includes things like touching files, etc.  CleanUp is
// always called exactly once, at the end of the task, regardless of success.
type Messy interface {
	CleanUp()
}

func revert(j db.Commiter, stscap *status.S) {
	if err := j.Rollback(); err != nil {
		if *stscap == nil {
			if s, ok := err.(status.S); ok {
				*stscap = s
			} else {
				*stscap = status.InternalError(err, "failed to rollback job after success")
			}
		} else {
			*stscap = status.WithSuppressed(*stscap, err)
		}
	}
}
