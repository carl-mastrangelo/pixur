package handlers

import (
	"database/sql"
	"log"
	"mime/multipart"
	"net/http"
	"time"

	"pixur.org/pixur/tasks"
)

type CreatePicHandler struct {
	// embeds
	http.Handler

	// deps
	DB      *sql.DB
	PixPath string
	Now     func() time.Time
}

func (h *CreatePicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rc := &requestChecker{r: r, now: h.Now}
	rc.checkPost()
	rc.checkXsrf()
	pwt := rc.getAuth()
	if rc.code != 0 {
		http.Error(w, rc.message, rc.code)
		return
	}

	ctx, err := addUserIDToCtx(r.Context(), pwt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var filename string
	var filedata multipart.File
	var fileURL string
	if uploadedFile, fileHeader, err := r.FormFile("file"); err != nil {
		if err != http.ErrMissingFile && err != http.ErrNotMultipart {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		filename = fileHeader.Filename
		filedata = uploadedFile
	}
	fileURL = r.FormValue("url")
	// TODO: close filedata?
	var task = &tasks.CreatePicTask{
		PixPath:  h.PixPath,
		DB:       h.DB,
		FileData: filedata,
		Filename: filename,
		FileURL:  fileURL,
		TagNames: r.PostForm["tag"],
		Ctx:      ctx,
	}

	runner := new(tasks.TaskRunner)
	if err := runner.Run(task); err != nil {
		returnTaskError(w, err)
		return
	}

	resp := CreatePicResponse{
		Pic: apiPic(task.CreatedPic),
	}

	returnProtoJSON(w, r, &resp)
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
