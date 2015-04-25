package pixur

import (
	"bytes"
	"crypto/sha512"
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"pixur.org/pixur/schema"
	"strings"
	"sync"
	"unicode"
)

var (
	errTagNotFound   = fmt.Errorf("Unable to find Tag")
	errDuplicateTags = fmt.Errorf("Data Corruption: Duplicate tags found")
	errInvalidFormat = fmt.Errorf("Unknown image format")
)

type readAtSeeker interface {
	io.Reader
	io.ReaderAt
	io.Seeker
}

type CreatePicTask struct {
	// Deps
	pixPath string
	db      *sql.DB

	// Inputs
	Filename string
	FileData readAtSeeker
	TagNames []string

	// Alternatively, a url can be uploaded
	FileURL string

	// State
	// The file that was created to hold the upload.
	tempFilename string
	tx           *sql.Tx

	// Results
	CreatedPic *schema.Pic
}

func (t *CreatePicTask) ResetForRetry() {
	t.reset()
}

func (t *CreatePicTask) CleanUp() {
	t.reset()
}

func (t *CreatePicTask) reset() {
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

	var p = new(schema.Pic)
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

	digest, err := getFileHash(wf)
	if err != nil {
		return err
	}
	p.Sha512Hash = digest

	img, err := FillImageConfig(wf, p)
	if err != nil {
		return err
	}
	thumbnail := MakeThumbnail(img)
	if err := t.beginTransaction(); err != nil {
		return err
	}

	if err := p.InsertAndSetId(t.tx); err != nil {
		return err
	}
	if err := t.renameTempFile(p); err != nil {
		return err
	}

	// If there is a problem creating the thumbnail, just continue on.
	if err := SaveThumbnail(thumbnail, p, t.pixPath); err != nil {
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
func (t *CreatePicTask) moveUploadedFile(tempFile io.Writer, p *schema.Pic) error {
	// TODO: check if the t.FileData is an os.File, and then try moving it.
	if bytesWritten, err := io.Copy(tempFile, t.FileData); err != nil {
		return err
	} else {
		p.FileSize = bytesWritten
	}
	// Attempt to flush the file incase an outside program needs to read from it.
	if f, ok := tempFile.(*os.File); ok {
		// If there was a failure, just give up.  The enclosing task will fail.
		if err := f.Sync(); err != nil {
			log.Println("Failed to sync file, continuing anwyays", err)
		}
	}
	return nil
}

func (t *CreatePicTask) downloadFile(tempFile io.Writer, p *schema.Pic) error {
	resp, err := http.Get(t.FileURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Failed to Download Pic %s [%d]", t.FileURL, resp.StatusCode)
	}

	if bytesWritten, err := io.Copy(tempFile, resp.Body); err != nil {
		return err
	} else {
		p.FileSize = bytesWritten
	}
	// Attempt to flush the file incase an outside program needs to read from it.
	if f, ok := tempFile.(*os.File); ok {
		// If there was a failure, just give up.  The enclosing task will fail.
		if err := f.Sync(); err != nil {
			log.Println("Failed to sync file, continuing anwyays", err)
		}
	}
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

func (t *CreatePicTask) renameTempFile(p *schema.Pic) error {
	if err := os.Rename(t.tempFilename, p.Path(t.pixPath)); err != nil {
		return err
	}
	// point this at the new file, incase the overall transaction fails
	t.tempFilename = p.Path(t.pixPath)
	return nil
}

// This function is not really transactional, because it hits multiple entity roots.
// TODO: test this.
func (t *CreatePicTask) insertOrFindTags() ([]*schema.Tag, error) {
	type findTagResult struct {
		tag *schema.Tag
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

			tag, err := findAndUpsertTag(name, now, t.tx)
			lock.Lock()
			defer lock.Unlock()
			resultMap[name] = &findTagResult{
				tag: tag,
				err: err,
			}
		}(tagName)
	}
	wg.Wait()

	var allTags []*schema.Tag
	for _, result := range resultMap {
		if result.err != nil {
			return nil, result.err
		}
		allTags = append(allTags, result.tag)
	}

	return allTags, nil
}

func getFileHash(f io.ReadSeeker) (string, error) {
	if _, err := f.Seek(0, os.SEEK_SET); err != nil {
		return "", err
	}
	defer f.Seek(0, os.SEEK_SET)
	h := sha512.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// findAndUpsertTag looks for an existing tag by name.  If it finds it, it updates the modified
// time and usage counter.  Otherwise, it creates a new tag with an initial count of 1.
func findAndUpsertTag(tagName string, now int64, tx *sql.Tx) (*schema.Tag, error) {
	tag, err := findTagByName(tagName, tx)
	if err == errTagNotFound {
		tag, err = createTag(tagName, now, tx)
	} else if err != nil {
		return nil, err
	} else {
		tag.ModifiedTime = now
		tag.Count += 1
		_, err = tag.Update(tx)
	}

	if err != nil {
		return nil, err
	}

	return tag, nil
}

func createTag(tagName string, now int64, tx *sql.Tx) (*schema.Tag, error) {
	tag := &schema.Tag{
		Name:         tagName,
		Count:        1,
		CreatedTime:  now,
		ModifiedTime: now,
	}

	if err := tag.InsertAndSetId(tx); err != nil {
		return nil, err
	}
	return tag, nil
}

func findTagByName(tagName string, tx *sql.Tx) (*schema.Tag, error) {
	stmt, err := schema.TagPrepare("SELECT * FROM_ WHERE %s = ? FOR UPDATE;", tx, schema.TagColName)
	if err != nil {
		return nil, err
	}

	tag, err := schema.LookupTag(stmt, tagName)
	if err == sql.ErrNoRows {
		return nil, errTagNotFound
	} else if err != nil {
		return nil, err
	}
	return tag, nil
}

func (t *CreatePicTask) addTagsForPic(p *schema.Pic, tags []*schema.Tag) error {
	for _, tag := range tags {
		picTag := &schema.PicTag{
			PicId:        p.Id,
			TagId:        tag.Id,
			Name:         tag.Name,
			CreatedTime:  p.CreatedTime,
			ModifiedTime: p.ModifiedTime,
		}
		if _, err := picTag.Insert(t.tx); err != nil {
			return err
		}
	}
	return nil
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

func fillTimestamps(p *schema.Pic) {
	p.CreatedTime = getNowMillis()
	p.ModifiedTime = p.CreatedTime
}
