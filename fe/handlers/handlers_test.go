package handlers

import (
	"pixur.org/pixur/api"
)

func init() {
	feconf := &feConfiguration{
		beconf: &api.BackendConfiguration{},
	}
	feconf.denorm()
	globalConfig.val.Store(feconf)
}
