package handlers

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"pixur.org/pixur/tasks"
)

type UpsertPicHandler struct {
	// embeds
	http.Handler

	// deps
	DB      *sql.DB
	PixPath string
	Now     func() time.Time
}

// TODO: add tests
func (h *UpsertPicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rc := &requestChecker{r: r, now: h.Now}
	rc.checkPost()
	rc.checkXsrf()
	rc.checkAuth()
	if rc.code != 0 {
		http.Error(w, rc.message, rc.code)
		return
	}

	var filename string
	var filedata multipart.File
	var fileURL string
	var md5Hash []byte
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
	if hexHash := r.FormValue("md5_hash"); hexHash != "" {
		md5Hash, err := hex.DecodeString(hexHash)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		} else if len(md5Hash) != md5.Size {
			http.Error(w, "Bad md5 hash", http.StatusBadRequest)
			return
		}
	}
	// TODO: close filedata?
	var task = &tasks.UpsertPicTask{
		PixPath:    h.PixPath,
		DB:         h.DB,
		HTTPClient: http.DefaultClient,
		TempFile:   ioutil.TempFile,
		Rename:     os.Rename,
		MkdirAll:   os.MkdirAll,
		Now:        h.Now,

		FileURL: fileURL,
		File:    filedata,
		Md5Hash: md5Hash,
		Header: tasks.FileHeader{
			Name: filename,
		},
		TagNames: r.PostForm["tag"],
	}

	runner := new(tasks.TaskRunner)
	if err := runner.Run(task); err != nil {
		returnTaskError(w, err)
		return
	}

	resp := UpsertPicResponse{
		Pic: apiPic(task.CreatedPic),
	}

	returnProtoJSON(w, r, &resp)
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
