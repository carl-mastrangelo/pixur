package pixur

import (
	"database/sql"
	"fmt"
	"image"
	"image/draw"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"os"
	"strings"

	"github.com/nfnt/resize"

	_ "image/gif"
	"image/jpeg"
	_ "image/png"
)

const (
	thumbnailWidth  = 120
	thumbnailHeight = 120
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
		if err := t.tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Println("Error in CreatePicTask", err)
		}
	}
}

func (t *CreatePicTask) Run() TaskError {
	wf, err := ioutil.TempFile(t.pixPath, "__")
	if err != nil {
		return err
	}
	defer wf.Close()
	t.tempFilename = wf.Name()

	var p = new(Pic)
	if err := t.moveUploadedFile(wf, p); err != nil {
		return err
	}
	img, err := t.fillImageConfig(wf, p)
	if err != nil {
		return err
	}
	thumbnail := makeThumbnail(img)
	if err := t.beginTransaction(); err != nil {
		return err
	}
	if err := t.insertPic(p); err != nil {
		return err
	}
	if err := t.renameTempFile(p); err != nil {
		return err
	}

	// If there is a problem creating the thumbnail, just continue on.
	if err := t.saveThumbnail(thumbnail, p); err != nil {
		log.Println("WARN Failed to create thumbnail", err)
	}

	if err := t.tx.Commit(); err != nil {
		return err
	}

	// The upload succeeded
	t.tempFilename = ""

	return nil
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

func (t *CreatePicTask) fillImageConfig(tempFile io.ReadSeeker, p *Pic) (image.Image, error) {
	if _, err := tempFile.Seek(0, os.SEEK_SET); err != nil {
		return nil, err
	}

	img, imgType, err := image.Decode(tempFile)

	if err != nil {
		return nil, err
	}

	// TODO: handle this error
	p.Mime, _ = FromImageFormat(imgType)
	p.Width = img.Bounds().Dx()
	p.Height = img.Bounds().Dy()
	return img, nil
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
	if err := os.Rename(t.tempFilename, p.Path(t.pixPath)); err != nil {
		return err
	}
	// point this at the new file, incase the overall transaction fails
	t.tempFilename = p.Path(t.pixPath)
	return nil
}

func (t *CreatePicTask) saveThumbnail(img image.Image, p *Pic) error {
	f, err := os.Create(p.ThumbnailPath(t.pixPath))
	if err != nil {
		return err
	}
	defer f.Close()
	return jpeg.Encode(f, img, nil)
}

func makeThumbnail(img image.Image) image.Image {
	bounds := findMaxSquare(img.Bounds())
	largeSquareImage := image.NewNRGBA(bounds)
	draw.Draw(largeSquareImage, bounds, img, bounds.Min, draw.Src)
	return resize.Resize(thumbnailWidth, thumbnailHeight, largeSquareImage, resize.NearestNeighbor)
}

func findMaxSquare(bounds image.Rectangle) image.Rectangle {
	width := bounds.Dx()
	height := bounds.Dy()
	if height < width {
		missingSpace := width - height
		return image.Rectangle{
			Min: image.Point{
				X: bounds.Min.X + missingSpace/2,
				Y: bounds.Min.Y,
			},
			Max: image.Point{
				X: bounds.Min.X + missingSpace/2 + height,
				Y: bounds.Max.Y,
			},
		}
	} else {
		missingSpace := height - width
		return image.Rectangle{
			Min: image.Point{
				X: bounds.Min.X,
				Y: bounds.Min.Y + missingSpace/2,
			},
			Max: image.Point{
				X: bounds.Max.X,
				Y: bounds.Min.Y + missingSpace/2 + width,
			},
		}
	}
}
