// Package tasks implements the core Pixur business logic.
package tasks // import "pixur.org/pixur/be/tasks"

import (
	"context"

	"pixur.org/pixur/be/schema/db"
	"pixur.org/pixur/be/status"
)

type Task interface {
	Run(context.Context) status.S
}

func revert(j db.Commiter, stscap *status.S) {
	if err := j.Rollback(); err != nil {
		status.ReplaceOrSuppress(stscap, status.From(err))
	}
}
