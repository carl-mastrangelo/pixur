package pixur

import (
	"mime/multipart"
	"net/http"
  "encoding/json"
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
		if err != http.ErrMissingFile {
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
	}
	defer task.Reset()

	if err := task.Run(); err != nil {
    return nil
  }
  
  w.Header().Set("Content-Type", "application/json")
  enc := json.NewEncoder(w)
 if err := enc.Encode(task.CreatedPic); err != nil {
		return err
	}

	return nil
}
