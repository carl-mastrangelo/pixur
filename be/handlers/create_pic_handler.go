package handlers

import (
	"bytes"
	"context"
	"mime/multipart"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/tasks"
)

func (s *serv) handleCreatePic(ctx context.Context, req *api.CreatePicRequest) (
	*api.CreatePicResponse, status.S) {
	var file multipart.File = &memFile{bytes.NewReader(req.FileData)}

	var task = &tasks.CreatePicTask{
		PixPath:  s.pixpath,
		DB:       s.db,
		FileData: file,
		Filename: req.FileName,
		FileURL:  req.FileUrl,
		TagNames: req.Tag,
		Ctx:      ctx,
	}

	if sts := s.runner.Run(task); sts != nil {
		return nil, sts
	}

	return &api.CreatePicResponse{
		Pic: apiPic(task.CreatedPic),
	}, nil
}
