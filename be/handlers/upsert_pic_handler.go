package handlers

import (
	"bytes"
	"context"
	"crypto/md5"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"

	"pixur.org/pixur/api"
	"pixur.org/pixur/status"
	"pixur.org/pixur/tasks"
)

var _ multipart.File = memFile{}

type memFile struct {
	*bytes.Reader
}

func (f memFile) Close() error {
	return nil
}

func (s *serv) handleUpsertPic(ctx context.Context, req *api.UpsertPicRequest) (*api.UpsertPicResponse, status.S) {

	var file multipart.File
	if len(req.Data) != 0 {
		// make sure this is non nil only if there is actually data.
		file = memFile{bytes.NewReader(req.Data)}
	}

	switch len(req.Md5Hash) {
	case 0:
	case md5.Size:
	default:
		return nil, status.InvalidArgument(nil, "bad md5 hash")
	}

	var task = &tasks.UpsertPicTask{
		PixPath:    s.pixpath,
		DB:         s.db,
		HTTPClient: http.DefaultClient,
		TempFile:   ioutil.TempFile,
		Rename:     os.Rename,
		MkdirAll:   os.MkdirAll,
		Now:        s.now,

		FileURL: req.Url,
		File:    file,
		Md5Hash: req.Md5Hash,
		Header: tasks.FileHeader{
			Name: req.Name,
		},
		TagNames: req.Tag,
		Ctx:      ctx,
	}

	if sts := s.runner.Run(task); sts != nil {
		return nil, sts
	}

	return &api.UpsertPicResponse{
		Pic: apiPic(task.CreatedPic),
	}, nil
}
