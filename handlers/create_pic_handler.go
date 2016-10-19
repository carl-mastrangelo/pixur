package handlers

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"time"

	"pixur.org/pixur/schema/db"
	"pixur.org/pixur/status"
	"pixur.org/pixur/tasks"
)

type CreatePicHandler struct {
	// embeds
	http.Handler

	// deps
	DB      db.DB
	PixPath string
	Now     func() time.Time
}

func (h *CreatePicHandler) CreatePic(ctx context.Context, req *CreatePicRequest) (
	*CreatePicResponse, status.S) {
	return h.createPic(ctx, req, nil)
}

func (h *CreatePicHandler) createPic(
	ctx context.Context, req *CreatePicRequest, file multipart.File) (
	*CreatePicResponse, status.S) {

	ctx, sts := fillUserIDFromCtx(ctx)
	if sts != nil {
		return nil, sts
	}

	if file == nil {
		file = &memFile{bytes.NewReader(req.FileData)}
	}

	var task = &tasks.CreatePicTask{
		PixPath:  h.PixPath,
		DB:       h.DB,
		FileData: file,
		Filename: req.FileName,
		FileURL:  req.FileUrl,
		TagNames: req.Tag,
		Ctx:      ctx,
	}

	var runner *tasks.TaskRunner
	if sts := runner.Run(task); sts != nil {
		return nil, sts
	}

	return &CreatePicResponse{
		Pic: apiPic(task.CreatedPic),
	}, nil
}

func (h *CreatePicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	if uploadedFile, fileHeader, err := r.FormFile("file"); err != nil {
		if err != http.ErrMissingFile && err != http.ErrNotMultipart {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		filename = fileHeader.Filename
		filedata = uploadedFile
	}

	resp, sts := h.createPic(ctx, &CreatePicRequest{
		FileName: filename,
		FileUrl:  r.FormValue("url"),
		Tag:      r.PostForm["tag"],
	}, filedata)
	if sts != nil {
		http.Error(w, sts.Message(), sts.Code().HttpStatus())
		return
	}

	returnProtoJSON(w, r, resp)
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/api/createPic", &CreatePicHandler{
			DB:      c.DB,
			PixPath: c.PixPath,
			Now:     time.Now,
		})
	})
}
