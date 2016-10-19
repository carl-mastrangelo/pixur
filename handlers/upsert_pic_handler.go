package handlers

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"pixur.org/pixur/schema/db"
	"pixur.org/pixur/status"
	"pixur.org/pixur/tasks"
)

type UpsertPicHandler struct {
	// embeds
	http.Handler

	// deps
	DB      db.DB
	PixPath string
	Now     func() time.Time
}

type memFile struct {
	*bytes.Reader
}

func (f *memFile) Close() error {
	return nil
}

func (h *UpsertPicHandler) upsertPic(
	ctx context.Context, req *UpsertPicRequest, file multipart.File) (*UpsertPicResponse, status.S) {

	ctx, sts := fillUserIDFromCtx(ctx)
	if sts != nil {
		return nil, sts
	}

	if file == nil {
		file = &memFile{bytes.NewReader(req.Data)}
	}

	switch len(req.Md5Hash) {
	case 0:
	case md5.Size:
	default:
		return nil, status.InvalidArgument(nil, "bad md5 hash")
	}

	var task = &tasks.UpsertPicTask{
		PixPath:    h.PixPath,
		DB:         h.DB,
		HTTPClient: http.DefaultClient,
		TempFile:   ioutil.TempFile,
		Rename:     os.Rename,
		MkdirAll:   os.MkdirAll,
		Now:        h.Now,

		FileURL: req.Url,
		File:    file,
		Md5Hash: req.Md5Hash,
		Header: tasks.FileHeader{
			Name: req.Name,
		},
		TagNames: req.Tag,
		Ctx:      ctx,
	}

	runner := new(tasks.TaskRunner)
	if sts := runner.Run(task); sts != nil {
		return nil, sts
	}

	return &UpsertPicResponse{
		Pic: apiPic(task.CreatedPic),
	}, nil
}

func (h *UpsertPicHandler) UpsertPic(
	ctx context.Context, req *UpsertPicRequest) (*UpsertPicResponse, status.S) {
	return h.upsertPic(ctx, req, nil)
}

// TODO: add tests
func (h *UpsertPicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rc := &requestChecker{r: r, now: h.Now}
	rc.checkPost()
	rc.checkXsrf()
	if rc.code != 0 {
		http.Error(w, rc.message, rc.code)
		return
	}

	ctx := r.Context()
	if token, present := authTokenFromReq(r); present {
		ctx = tasks.CtxFromAuthToken(ctx, token)
	}

	var filename string
	var filedata multipart.File
	if uploadedFile, fileHeader, err := r.FormFile("data"); err != nil {
		if err != http.ErrMissingFile && err != http.ErrNotMultipart {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
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
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	resp, sts := h.upsertPic(ctx, &UpsertPicRequest{
		Url:     r.FormValue("url"),
		Name:    filename,
		Md5Hash: md5Hash,
		Tag:     r.PostForm["tag"],
	}, filedata)

	if sts != nil {
		http.Error(w, sts.Message(), sts.Code().HttpStatus())
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
