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
	"net/http"
	"os"
	"sync"
	"time"

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
	TagNames []string

	// Alternatively, a url can be uploaded
	FileURL string

	// State
	// The file that was created to hold the upload.
	tempFilename string
	tx           *sql.Tx

	// Results
	CreatedPic *Pic
}

func (t *CreatePicTask) Reset() {
	if t.tempFilename != "" {
		if err := os.Remove(t.tempFilename); err != nil {
			log.Println("ERROR Unable to remove image in CreatePicTask", t.tempFilename, err)
		}
	}
	if t.tx != nil {
		if err := t.tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Println("ERROR Unable to rollback in CreatePicTask", err)
		}
	}
}

func (t *CreatePicTask) Run() TaskError {
	wf, err := ioutil.TempFile(t.pixPath, "__")
	if err != nil {
		return WrapError(err)
	}
	defer wf.Close()
	t.tempFilename = wf.Name()

	var p = new(Pic)
	fillTimestamps(p)

	if t.FileData != nil {
		if err := t.moveUploadedFile(wf, p); err != nil {
			return err
		}
	} else if t.FileURL != "" {
		if err := t.downloadFile(wf, p); err != nil {
			return err
		}
	} else {
		return WrapError(fmt.Errorf("No file uploaded"))
	}

	if _, err := t.insertOrFindTags(); err != nil {
		return err
	}

	img, err := t.fillImageConfig(wf, p)
	if err != nil {
		return WrapError(err)
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
		return WrapError(err)
	}

	// The upload succeeded
	t.tempFilename = ""
	t.CreatedPic = p
	return nil
}

// Moves the uploaded file and records the file size.  It might not be possible to just move the
// file in the event that the uploaded location is on a different partition than persistent dir.
func (t *CreatePicTask) moveUploadedFile(tempFile io.Writer, p *Pic) TaskError {
	// TODO: check if the t.FileData is an os.File, and then try moving it.
	if bytesWritten, err := io.Copy(tempFile, t.FileData); err != nil {
		return WrapError(err)
	} else {
		p.FileSize = bytesWritten
	}
	return nil
}

func (t *CreatePicTask) downloadFile(tempFile io.Writer, p *Pic) TaskError {
	resp, err := http.Get(t.FileURL)
	if err != nil {
		return WrapError(err)
	}
	defer resp.Body.Close()
	// TODO: check the response code

	if bytesWritten, err := io.Copy(tempFile, resp.Body); err != nil {
		return WrapError(err)
	} else {
		p.FileSize = bytesWritten
	}
	return nil
}

func (t *CreatePicTask) fillImageConfig(tempFile io.ReadSeeker, p *Pic) (image.Image, TaskError) {
	if _, err := tempFile.Seek(0, os.SEEK_SET); err != nil {
		return nil, WrapError(err)
	}

	img, imgType, err := image.Decode(tempFile)

	if err != nil {
		return nil, WrapError(err)
	}

	// TODO: handle this error
	p.Mime, _ = FromImageFormat(imgType)
	p.Width = int64(img.Bounds().Dx())
	p.Height = int64(img.Bounds().Dy())
	return img, nil
}

func (t *CreatePicTask) beginTransaction() TaskError {
	if tx, err := t.db.Begin(); err != nil {
		return WrapError(err)
	} else {
		t.tx = tx
	}
	return nil
}

func (t *CreatePicTask) insertPic(p *Pic) TaskError {
	res, err := t.tx.Exec(p.BuildInsert(), p.ColumnPointers(p.GetColumnNames())...)
	if err != nil {
		return WrapError(err)
	}
	if insertId, err := res.LastInsertId(); err != nil {
		return WrapError(err)
	} else {
		p.Id = insertId
	}
	return nil
}

func (t *CreatePicTask) renameTempFile(p *Pic) TaskError {
	if err := os.Rename(t.tempFilename, p.Path(t.pixPath)); err != nil {
		return WrapError(err)
	}
	// point this at the new file, incase the overall transaction fails
	t.tempFilename = p.Path(t.pixPath)
	return nil
}

func (t *CreatePicTask) saveThumbnail(img image.Image, p *Pic) TaskError {
	f, err := os.Create(p.ThumbnailPath(t.pixPath))
	if err != nil {
		return WrapError(err)
	}
	defer f.Close()
	return WrapError(jpeg.Encode(f, img, nil))
}

// This function is not really transactional, because it hits multiple entity roots.
// TODO: break this apart, test it.
func (t *CreatePicTask) insertOrFindTags() ([]*Tag, TaskError) {
	var wg sync.WaitGroup
	type findTagResult struct {
		tag *Tag
		err error
	}

	var resultMap = make(map[string]*findTagResult, len(t.TagNames))
	var lock sync.Mutex

	wg.Add(len(t.TagNames))
	for _, tagName := range t.TagNames {
		go func(name string) {
			defer wg.Done()
			tags, err := findTags(t.db, "SELECT * FROM tags WHERE name = ?;", name)
			var res *findTagResult
			if err != nil {
				res = &findTagResult{
					err: err,
				}
			} else if len(tags) == 0 {
				res = &findTagResult{}
			} else if len(tags) == 1 {
				res = &findTagResult{
					tag: tags[0],
				}
			} else {
				res = &findTagResult{
					err: fmt.Errorf("Data corruption: multiple tags: %v", tags),
				}
			}
			lock.Lock()
			defer lock.Unlock()
			resultMap[name] = res
		}(tagName)
	}

	wg.Wait()

	for tagName, tagResult := range resultMap {
		if tagResult.err != nil {
			return nil, WrapError(tagResult.err)
		} else if tagResult.tag == nil {
			wg.Add(1)
			go func(name string, result *findTagResult) {
				defer wg.Done()
				now := getNowMillis()
				tag := &Tag{
					Name:         name,
					CreatedTime:  now,
					ModifiedTime: now,
				}

				res, err := t.db.Exec(tag.BuildInsert(), tag.ColumnPointers(tag.GetColumnNames())...)
				if err != nil {
					result.err = err
					return
				}
				if insertId, err := res.LastInsertId(); err != nil {
					result.err = err
					return
				} else {
					tag.Id = insertId
				}
				result.tag = tag
			}(tagName, tagResult)
		}
	}
	wg.Wait()

	var allTags []*Tag
	for tagName, result := range resultMap {
		fmt.Printf("%v: %+v, %v", tagName, result.tag, result.err)
		if result.err != nil {
			return nil, WrapError(result.err)
		}
		allTags = append(allTags, result.tag)
	}

	return allTags, nil
}

func findTags(db *sql.DB, query string, args ...interface{}) ([]*Tag, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	columnNames, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var tags []*Tag
	for rows.Next() {
		t := new(Tag)
		if err := rows.Scan(t.ColumnPointers(columnNames)...); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tags, nil
}

// TODO: interpret image rotation metadata
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

func getNowMillis() int64 {
	return int64(time.Duration(time.Now().UnixNano()) / time.Millisecond)
}

func fillTimestamps(p *Pic) {
	p.CreatedTime = getNowMillis()
	p.ModifiedTime = p.CreatedTime
}
