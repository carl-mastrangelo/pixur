package pixur

import (
	"bytes"
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
	"strings"
	"sync"
	"unicode"

	"github.com/nfnt/resize"

	_ "image/gif"
	"image/jpeg"
	_ "image/png"
)

const (
	thumbnailWidth  = 160
	thumbnailHeight = 160
)

var (
	errTagNotFound   = fmt.Errorf("Unable to find Tag")
	errDuplicateTags = fmt.Errorf("Data Corruption: Duplicate tags found")
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

func (t *CreatePicTask) Run() error {
	wf, err := ioutil.TempFile(t.pixPath, "__")
	if err != nil {
		return err
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
		return fmt.Errorf("No file uploaded")
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

	tags, err := t.insertOrFindTags()
	if err != nil {
		return err
	}
	// must happen after pic is created, because it depends on pic id
	if err := t.addTagsForPic(p, tags); err != nil {
		return err
	}

	if err := t.tx.Commit(); err != nil {
		return err
	}

	// The upload succeeded
	t.tempFilename = ""
	t.CreatedPic = p
	return nil
}

// Moves the uploaded file and records the file size.  It might not be possible to just move the
// file in the event that the uploaded location is on a different partition than persistent dir.
func (t *CreatePicTask) moveUploadedFile(tempFile io.Writer, p *Pic) error {
	// TODO: check if the t.FileData is an os.File, and then try moving it.
	if bytesWritten, err := io.Copy(tempFile, t.FileData); err != nil {
		return err
	} else {
		p.FileSize = bytesWritten
	}
	return nil
}

func (t *CreatePicTask) downloadFile(tempFile io.Writer, p *Pic) error {
	resp, err := http.Get(t.FileURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// TODO: check the response code

	if bytesWritten, err := io.Copy(tempFile, resp.Body); err != nil {
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
	p.Width = int64(img.Bounds().Dx())
	p.Height = int64(img.Bounds().Dy())
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
	res, err := t.tx.Exec(p.BuildInsert(), p.ColumnPointers(p.GetColumnNames())...)
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

// This function is not really transactional, because it hits multiple entity roots.
// TODO: test this.
func (t *CreatePicTask) insertOrFindTags() ([]*Tag, error) {
	type findTagResult struct {
		tag *Tag
		err error
	}

	var cleanedTags = cleanTagNames(t.TagNames)

	var resultMap = make(map[string]*findTagResult, len(cleanedTags))
	var lock sync.Mutex

	now := getNowMillis()

	var wg sync.WaitGroup
	for _, tagName := range cleanedTags {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()

			tx, err := t.db.Begin()
			if err != nil {
				lock.Lock()
				defer lock.Unlock()
				resultMap[name] = &findTagResult{
					tag: nil,
					err: err,
				}
				return
			}

			tag, err := findAndUpsertTag(name, now, tx)
			if err != nil {
				// TODO: maybe do something with this error?
				tx.Rollback()
			} else {
				err = tx.Commit()
			}

			lock.Lock()
			defer lock.Unlock()
			resultMap[name] = &findTagResult{
				tag: tag,
				err: err,
			}
		}(tagName)
	}
	wg.Wait()

	var allTags []*Tag
	for _, result := range resultMap {
		if result.err != nil {
			return nil, result.err
		}
		allTags = append(allTags, result.tag)
	}

	return allTags, nil
}

// findAndUpsertTag looks for an existing tag by name.  If it finds it, it updates the modified
// time and usage counter.  Otherwise, it creates a new tag with an initial count of 1.
func findAndUpsertTag(tagName string, now int64, tx *sql.Tx) (*Tag, error) {
	tag, err := findTagByName(tagName, tx)
	if err == errTagNotFound {
		tag, err = createTag(tagName, now, tx)
	} else if err != nil {
		return nil, err
	} else {
		tag.ModifiedTime = now
		tag.Count += 1
		err = tag.Update(tx)
	}

	if err != nil {
		return nil, err
	}

	return tag, nil
}

func createTag(tagName string, now int64, tx *sql.Tx) (*Tag, error) {
	tag := &Tag{
		Name:         tagName,
		Count:        1,
		CreatedTime:  now,
		ModifiedTime: now,
	}

	if err := tag.Insert(tx); err != nil {
		return nil, err
	}
	return tag, nil
}

func findTagByName(tagName string, tx *sql.Tx) (*Tag, error) {
	tags, err := findTags(tx, "SELECT * FROM tags WHERE name = ?;", tagName)
	if err != nil {
		return nil, err
	}
	switch len(tags) {
	case 0:
		return nil, errTagNotFound
	case 1:
		return tags[0], nil
	default:
		return nil, errDuplicateTags
	}
}

func findTags(tx *sql.Tx, query string, args ...interface{}) ([]*Tag, error) {
	rows, err := tx.Query(query, args...)
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

func findPicTagsByPicId(picId int64, db *sql.DB) ([]*PicTag, error) {
	return findPicTags(db, "SELECT * FROM pictags WHERE pic_id = ?;", picId)
}

func findPicTags(db *sql.DB, query string, args ...interface{}) ([]*PicTag, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	columnNames, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var picTags []*PicTag
	for rows.Next() {
		pt := new(PicTag)
		if err := rows.Scan(pt.ColumnPointers(columnNames)...); err != nil {
			return nil, err
		}
		picTags = append(picTags, pt)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return picTags, nil
}

func (t *CreatePicTask) addTagsForPic(p *Pic, tags []*Tag) error {
	for _, tag := range tags {
		picTag := &PicTag{
			PicId:        p.Id,
			TagId:        tag.Id,
			Name:         tag.Name,
			CreatedTime:  p.CreatedTime,
			ModifiedTime: p.ModifiedTime,
		}
		_, err := t.db.Exec(picTag.BuildInsert(), picTag.ColumnPointers(picTag.GetColumnNames())...)
		if err != nil {
			return err
		}
	}
	return nil
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

func cleanTagNames(rawTagNames []string) []string {
	var trimmed []string
	for _, tagName := range rawTagNames {
		trimmed = append(trimmed, strings.TrimSpace(tagName))
	}

	var noInvalidRunes []string
	for _, tagName := range trimmed {
		var buf bytes.Buffer
		for _, runeValue := range tagName {
			if runeValue == unicode.ReplacementChar || !unicode.IsPrint(runeValue) {
				continue
			}
			buf.WriteRune(runeValue)
		}
		noInvalidRunes = append(noInvalidRunes, buf.String())
	}

	// We keep track of which are duplicates, but maintain order otherwise
	var seen = make(map[string]struct{}, len(noInvalidRunes))

	var uniqueNonEmptyTags []string
	for _, tagName := range noInvalidRunes {
		if len(tagName) == 0 {
			continue
		}
		if _, present := seen[tagName]; present {
			continue
		}
		seen[tagName] = struct{}{}
		uniqueNonEmptyTags = append(uniqueNonEmptyTags, tagName)
	}

	return uniqueNonEmptyTags
}

func fillTimestamps(p *Pic) {
	p.CreatedTime = getNowMillis()
	p.ModifiedTime = p.CreatedTime
}
