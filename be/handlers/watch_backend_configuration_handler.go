package handlers

import (
	"pixur.org/pixur/api"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/tasks"
)

func (s *serv) handleWatchBackendConfiguration(
	req *api.WatchBackendConfigurationRequest,
	wbcs api.PixurService_WatchBackendConfigurationServer) status.S {

	ctx := wbcs.Context()

	beconf, sts := tasks.GetConfiguration(ctx)
	if sts != nil {
		return sts
	}
	resp := &api.WatchBackendConfigurationResponse{
		Version:              0, // TODO: implement
		BackendConfiguration: apiConfig(beconf),
	}

	if err := wbcs.Send(resp); err != nil {
		return status.Unavailable(err, "can't send config")
	}

	select {
	case <-ctx.Done():
		return status.From(ctx.Err())
	}

	return nil
}
