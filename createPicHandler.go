package pixur

import (
	"net/http"
)

func (s *Server) uploadHandler(w http.ResponseWriter, r *http.Request) error {
	if r.Method != "POST" {
		http.Error(w, "Unsupported Method", http.StatusMethodNotAllowed)
		return nil
	}

	uploadedFile, fileHeader, err := r.FormFile("file")
	if err == http.ErrMissingFile {
		http.Error(w, "Missing File", http.StatusBadRequest)
		return nil
	} else if err != nil {
		return err
	}

	var task = &CreatePicTask{
		pixPath:  s.pixPath,
		db:       s.db,
		FileData: uploadedFile,
		Filename: fileHeader.Filename,
	}
	defer task.Reset()

	return task.Run()
}
