package handlers

import (
	"bytes"
	"context"
	"mime/multipart"

	"pixur.org/pixur/api"
	"pixur.org/pixur/status"
	"pixur.org/pixur/tasks"
)

func (s *serv) handleCreatePic(ctx context.Context, req *api.CreatePicRequest) (
	*api.CreatePicResponse, status.S) {

	ctx, sts := fillUserIDFromCtx(ctx)
	if sts != nil {
		return nil, sts
	}

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

/*
func (h *CreatePicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	resp, sts := h.createPic(ctx, &api.CreatePicRequest{
		FileName: filename,
		FileUrl:  r.FormValue("url"),
		Tag:      r.PostForm["tag"],
	}, filedata)
	if sts != nil {
		httpError(w, sts)
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
*/
