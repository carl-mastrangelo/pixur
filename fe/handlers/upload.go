package handlers

import (
	"encoding/hex"
	"io/ioutil"
	"mime/multipart"
	"net/http"

	"pixur.org/pixur/api"
	"pixur.org/pixur/fe/server"
)

type upsertPicHandler struct {
	c  api.PixurServiceClient
	pt paths
}

func (h *upsertPicHandler) upsert(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var filename string
	var filedata multipart.File
	if uploadedFile, fileHeader, err := r.FormFile(h.pt.pr.File()); err != nil {
		if err != http.ErrMissingFile && err != http.ErrNotMultipart {
			httpError(w, &HTTPErr{
				Message: "can't read file: " + err.Error(),
				Code:    http.StatusBadRequest,
			})
			return
		}
	} else {
		filename = fileHeader.Filename
		filedata = uploadedFile
		defer filedata.Close()
	}

	var md5Hash []byte
	if hexHash := r.FormValue(h.pt.pr.Md5Hash()); hexHash != "" {
		var err error
		md5Hash, err = hex.DecodeString(hexHash)
		if err != nil {
			httpError(w, &HTTPErr{
				Message: "can't decode md5 hash: " + err.Error(),
				Code:    http.StatusBadRequest,
			})
			return
		}
	}
	var data []byte
	if filedata != nil {
		var err error
		data, err = ioutil.ReadAll(filedata)
		if err != nil {
			httpError(w, &HTTPErr{
				Message: "can't read file: " + err.Error(),
				Code:    http.StatusInternalServerError,
			})
			return
		}
	}

	resp, sts := h.c.UpsertPic(ctx, &api.UpsertPicRequest{
		Url:     r.FormValue(h.pt.pr.Url()),
		Name:    filename,
		Data:    data,
		Md5Hash: md5Hash,
		Tag:     r.PostForm[h.pt.pr.Tag()],
	})

	if sts != nil {
		httpError(w, sts)
		return
	}

	http.Redirect(w, r, h.pt.Viewer(resp.Pic.Id).RequestURI(), http.StatusSeeOther)
}

func init() {
	register(func(s *server.Server) error {
		h := upsertPicHandler{
			c:  s.Client,
			pt: paths{r: s.HTTPRoot},
		}

		s.HTTPMux.Handle(h.pt.UpsertPicAction().Path, newActionHandler(s, http.HandlerFunc(h.upsert)))
		return nil
	})
}
