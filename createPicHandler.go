package pixur

import (
  "net/http"
  "fmt"
  "os"
  "path/filepath"
  "io/ioutil"
  "io"
  "strings"
  
  "image"
  
  _ "image/jpeg"
  _ "image/gif"
  _ "image/png"
)


func (s *Server) uploadHandler(w http.ResponseWriter, r *http.Request) error {
  var p Pic
  if r.Method != "POST" {
    http.Error(w, "Unsupported Method", http.StatusMethodNotAllowed)
    return nil
  }

  rf, fh, err := r.FormFile("file")
  if err == http.ErrMissingFile {
    http.Error(w, "Missing File", http.StatusBadRequest)
    return nil
  } else if err != nil {
    return err
  }
  defer rf.Close()

  for k, v := range fh.Header {
    fmt.Println(k, v)
  }

  wf, err := ioutil.TempFile(s.pixPath, "tmp")
  if err != nil {
    return err
  }
  defer wf.Close()
  
  if bytesWritten, err := io.Copy(wf, rf); err != nil {
    return err
  } else {
    p.FileSize = bytesWritten
  }
  
  if _, err := wf.Seek(0, os.SEEK_SET); err != nil {
    return err
  }
  imageConfig, imageType, err := image.DecodeConfig(wf)
  if err != nil {
    return err
  }
  
  p.Mime, _ = FromImageFormat(imageType)
  p.Width = imageConfig.Width
  p.Height = imageConfig.Height

  tx, err := s.db.Begin()
  if err != nil {
    return err
  }
  
  var columnNames []string
  var columnValues []interface{}
  var valueFormat []string
  for name, value := range p.PointerMap() {
    columnNames = append(columnNames, name)
    columnValues = append(columnValues, value)
    valueFormat = append(valueFormat, "?")
  }
  
  stmt := fmt.Sprintf("INSERT INTO pix (%s) VALUES (%s);", 
      strings.Join(columnNames, ", "), strings.Join(valueFormat, ", "))
  
  res, err := tx.Exec(stmt, columnValues...)
  if err != nil {
    return err
  }
  if insertId, err := res.LastInsertId(); err != nil {
    return err
  } else {
    p.Id = insertId
  }
  
  newName := filepath.Join(s.pixPath, fmt.Sprintf("%d.%s", p.Id, p.Mime.Ext()))
  if err := os.Rename(wf.Name(), newName); err != nil {
    return err    
  }

  return tx.Commit()
}


