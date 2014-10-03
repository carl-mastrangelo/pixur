package pixur

import (
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"image"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

type CreatePicTask struct {
	// Deps
	pixPath string
	db      *sql.DB

	// Inputs
	Filename string
	FileData multipart.File

	// State
	// The file that was created to hold the upload.
	tempFilename string
	tx           *sql.Tx

	// Results
}

func (t *CreatePicTask) Reset() {
	if t.tempFilename != "" {
		if err := os.Remove(t.tempFilename); err != nil {
			log.Println("Error in CreatePicTask", err)
		}
	}
	if t.tx != nil {
		if err := t.tx.Rollback(); err != nil {
			log.Println("Error in CreatePicTask", err)
		}
	}
}

func (t *CreatePicTask) Run() TaskError {
	wf, err := ioutil.TempFile(t.pixPath, "tmp")
	if err != nil {
		return err
	}
	defer wf.Close()
	t.tempFilename = wf.Name()

	var p = new(Pic)
	if err := t.moveUploadedFile(wf, p); err != nil {
		return err
	}
	if err := t.fillImageConfig(wf, p); err != nil {
		return err
	}
	if err := t.beginTransaction(); err != nil {
		return err
	}
	if err := t.insertPic(p); err != nil {
		return err
	}
	if err := t.renameTempFile(p); err != nil {
		return err
	}

	return t.tx.Commit()
}

// Moves the uploaded file and records the file size
func (t *CreatePicTask) moveUploadedFile(tempFile io.Writer, p *Pic) error {
	// TODO: check if the t.FileData is an os.File, and then try moving it.
	if bytesWritten, err := io.Copy(tempFile, t.FileData); err != nil {
		return err
	} else {
		p.FileSize = bytesWritten
	}
	return nil
}

func (t *CreatePicTask) fillImageConfig(tempFile io.ReadSeeker, p *Pic) error {
	if _, err := tempFile.Seek(0, os.SEEK_SET); err != nil {
		return err
	}
	imageConfig, imageType, err := image.DecodeConfig(tempFile)
	if err != nil {
		return err
	}

	// TODO: handle this error
	p.Mime, _ = FromImageFormat(imageType)
	p.Width = imageConfig.Width
	p.Height = imageConfig.Height
	return nil
}

func (t *CreatePicTask) beginTransaction() error {
	if tx, err := t.db.Begin(); err != nil {
		return err
	} else {
		t.tx = tx
	}
	return nil
}

func (t *CreatePicTask) insertPic(p *Pic) error {
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

	res, err := t.tx.Exec(stmt, columnValues...)
	if err != nil {
		return err
	}
	if insertId, err := res.LastInsertId(); err != nil {
		return err
	} else {
		p.Id = insertId
	}
	return nil
}

func (t *CreatePicTask) renameTempFile(p *Pic) error {
	newName := filepath.Join(t.pixPath, fmt.Sprintf("%d.%s", p.Id, p.Mime.Ext()))
	if err := os.Rename(t.tempFilename, newName); err != nil {
		return err
	}
	// point this at the new file, incase the overall transaction fails
	t.tempFilename = newName
	return nil
}
