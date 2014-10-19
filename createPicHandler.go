package pixur

import (
	"encoding/json"
	"mime/multipart"
	"net/http"
)

func (s *Server) uploadHandler(w http.ResponseWriter, r *http.Request) error {
	if r.Method != "POST" {
		http.Error(w, "Unsupported Method", http.StatusMethodNotAllowed)
		return nil
	}

	var filename string
	var filedata multipart.File
	var fileURL string
	if uploadedFile, fileHeader, err := r.FormFile("file"); err != nil {
		if err != http.ErrMissingFile && err != http.ErrNotMultipart {
			return err
		}
	} else {
		filename = fileHeader.Filename
		filedata = uploadedFile
	}
	fileURL = r.FormValue("url")

	var task = &CreatePicTask{
		pixPath:  s.pixPath,
		db:       s.db,
		FileData: filedata,
		Filename: filename,
		FileURL:  fileURL,
		TagNames: r.PostForm["tag"],
	}
	defer task.Reset()

	if err := task.Run(); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	if err := enc.Encode(task.CreatedPic.ToInterface()); err != nil {
		return err
	}

	return nil
}
