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

type memFile struct {
	*bytes.Reader
}

func (f *memFile) Close() error {
	return nil
}

func (s *serv) handleUpsertPic(ctx context.Context, req *api.UpsertPicRequest) (*api.UpsertPicResponse, status.S) {

	ctx, sts := fillUserIDFromCtx(ctx)
	if sts != nil {
		return nil, sts
	}

	var file multipart.File
	if len(req.Data) != 0 {
		// make sure this is non nil only if there actually data.
		file = &memFile{bytes.NewReader(req.Data)}
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

// TODO: add tests
/*
func (h *UpsertPicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rc := &requestChecker{r: r, now: h.Now}
	rc.checkPost()
	rc.checkXsrf()
	if rc.sts != nil {
		httpError(w, rc.sts)
		return
	}

	ctx := r.Context()
	if token, present := authTokenFromReq(r); present {
		ctx = tasks.CtxFromAuthToken(ctx, token)
	}

	var filename string
	var filedata multipart.File
	if uploadedFile, fileHeader, err := r.FormFile("file"); err != nil {
		if err != http.ErrMissingFile && err != http.ErrNotMultipart {
			httpError(w, status.InvalidArgument(err, "can't read file"))
			return
		}
	} else {
		filename = fileHeader.Filename
		filedata = uploadedFile
	}

	var md5Hash []byte
	if hexHash := r.FormValue("md5_hash"); hexHash != "" {
		var err error
		md5Hash, err = hex.DecodeString(hexHash)
		if err != nil {
			httpError(w, status.InvalidArgument(err, "can't decode md5 hash"))
			return
		}
	}

	resp, sts := h.upsertPic(ctx, &api.UpsertPicRequest{
		Url:     r.FormValue("url"),
		Name:    filename,
		Md5Hash: md5Hash,
		Tag:     r.PostForm["tag"],
	}, filedata)

	if sts != nil {
		httpError(w, sts)
		return
	}

	returnProtoJSON(w, r, resp)
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/api/upsertPic", &UpsertPicHandler{
			DB:      c.DB,
			PixPath: c.PixPath,
			Now:     time.Now,
		})
	})
}
*/
