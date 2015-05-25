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
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
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
	now          time.Time

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
	t.now = time.Now()
	wf, err := ioutil.TempFile(t.pixPath, "__")
	if err != nil {
		return InternalError("Unable to create tempfile", err)
	}
	defer wf.Close()
	t.tempFilename = wf.Name()

	var p = new(schema.Pic)
	p.SetCreatedTime(t.now)
	p.SetModifiedTime(t.now)

	if t.FileData != nil {
		if err := t.moveUploadedFile(wf, p); err != nil {
			return err
		}
	} else if t.FileURL != "" {
		if err := t.downloadFile(wf, p); err != nil {
			return err
		}
	} else {
		return InvalidArgument("No file uploaded", nil)
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

	if err := p.Insert(t.tx); err != nil {
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
func (t *CreatePicTask) moveUploadedFile(tempFile io.Writer, p *schema.Pic) Status {
	// TODO: check if the t.FileData is an os.File, and then try moving it.
	if bytesWritten, err := io.Copy(tempFile, t.FileData); err != nil {
		return InternalError("Unable to move uploaded file", err)
	} else {
		p.FileSize = bytesWritten
	}
	// Attempt to flush the file incase an outside program needs to read from it.
	if f, ok := tempFile.(*os.File); ok {
		if err := f.Sync(); err != nil {
			return InternalError("Failed to sync uploaded file", err)
		}
	}
	return nil
}

func (t *CreatePicTask) downloadFile(tempFile io.Writer, p *schema.Pic) Status {
	resp, err := http.Get(t.FileURL)
	if err != nil {
		return InvalidArgument("Unable to download "+t.FileURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		message := fmt.Sprintf("Failed to Download Pic %s [%d]", t.FileURL, resp.StatusCode)
		return InvalidArgument(message, nil)
	}

	if bytesWritten, err := io.Copy(tempFile, resp.Body); err != nil {
		return InternalError("Failed to copy downloaded file", err)
	} else {
		p.FileSize = bytesWritten
	}
	// Attempt to flush the file incase an outside program needs to read from it.
	if f, ok := tempFile.(*os.File); ok {
		if err := f.Sync(); err != nil {
			return InternalError("Failed to sync file", err)
		}
	}
	return nil
}

func (t *CreatePicTask) beginTransaction() Status {
	if tx, err := t.db.Begin(); err != nil {
		return InternalError("Unable to Begin TX", err)
	} else {
		t.tx = tx
	}
	return nil
}

func (t *CreatePicTask) renameTempFile(p *schema.Pic) Status {
	if err := os.Rename(t.tempFilename, p.Path(t.pixPath)); err != nil {
		message := fmt.Sprintf("Unable to move uploaded file %s -> %s",
			t.tempFilename, p.Path(t.pixPath))
		return InternalError(message, err)
	}
	// point this at the new file, incase the overall transaction fails
	t.tempFilename = p.Path(t.pixPath)
	return nil
}

// This function is not really transactional, because it hits multiple entity roots.
// TODO: test this.
func (t *CreatePicTask) insertOrFindTags() ([]*schema.Tag, error) {
	cleanedTags, err := cleanTagNames(t.TagNames)
	if err != nil {
		return nil, err
	}
	sort.Strings(cleanedTags)
	var allTags []*schema.Tag
	for _, tagName := range cleanedTags {
		tag, err := findAndUpsertTag(tagName, t.now, t.tx)
		if err != nil {
			return nil, err
		}
		allTags = append(allTags, tag)
	}

	return allTags, nil
}

func getFileHash(f io.ReadSeeker) (string, Status) {
	if _, err := f.Seek(0, os.SEEK_SET); err != nil {
		return "", InternalError(err.Error(), err)
	}
	defer f.Seek(0, os.SEEK_SET)
	h := sha512.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", InternalError(err.Error(), err)
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// findAndUpsertTag looks for an existing tag by name.  If it finds it, it updates the modified
// time and usage counter.  Otherwise, it creates a new tag with an initial count of 1.
func findAndUpsertTag(tagName string, now time.Time, tx *sql.Tx) (*schema.Tag, error) {
	tag, err := findTagByName(tagName, tx)
	if err == errTagNotFound {
		tag, err = createTag(tagName, now, tx)
	} else if err != nil {
		return nil, err
	} else {
		tag.SetModifiedTime(now)
		tag.UsageCount += 1
		err = tag.Update(tx)
	}

	if err != nil {
		return nil, err
	}

	return tag, nil
}

func createTag(tagName string, now time.Time, tx *sql.Tx) (*schema.Tag, error) {
	tag := &schema.Tag{
		Name:       tagName,
		UsageCount: 1,
	}
	tag.SetCreatedTime(now)
	tag.SetModifiedTime(now)

	if err := tag.Insert(tx); err != nil {
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
			PicId: p.PicId,
			TagId: tag.TagId,
			Name:  tag.Name,
		}
		picTag.SetCreatedTime(p.GetCreatedTime())
		picTag.SetModifiedTime(p.GetModifiedTime())
		if _, err := picTag.Insert(t.tx); err != nil {
			return err
		}
	}
	return nil
}

func checkValidUnicode(tagNames []string) Status {
	for _, tn := range tagNames {
		if !utf8.ValidString(tn) {
			return InvalidArgument("Invalid tag name: "+tn, nil)
		}
	}
	return nil
}

func removeUnprintableCharacters(tagNames []string) []string {
	printableTagNames := make([]string, 0, len(tagNames))
	for _, tn := range tagNames {
		var buf bytes.Buffer
		buf.Grow(len(tn))
		for _, runeValue := range tn {
			if unicode.IsPrint(runeValue) {
				buf.WriteRune(runeValue)
			}
		}
		printableTagNames = append(printableTagNames, buf.String())
	}
	return printableTagNames
}

func trimTagNames(tagNames []string) []string {
	trimmed := make([]string, 0, len(tagNames))
	for _, tn := range tagNames {
		trimmed = append(trimmed, strings.TrimSpace(tn))
	}
	return trimmed
}

// removeDuplicateTagNames preserves order of the tags
func removeDuplicateTagNames(tagNames []string) []string {
	var seen = make(map[string]struct{}, len(tagNames))
	uniqueTagNames := make([]string, 0, len(tagNames))
	for _, tn := range tagNames {
		if _, present := seen[tn]; !present {
			seen[tn] = struct{}{}
			uniqueTagNames = append(uniqueTagNames, tn)
		}
	}
	return uniqueTagNames
}

func removeEmptyTagNames(tagNames []string) []string {
	nonEmptyTagNames := make([]string, 0, len(tagNames))
	for _, tn := range tagNames {
		if tn != "" {
			nonEmptyTagNames = append(nonEmptyTagNames, tn)
		}
	}
	return nonEmptyTagNames
}

func cleanTagNames(rawTagNames []string) ([]string, Status) {
	if err := checkValidUnicode(rawTagNames); err != nil {
		return nil, err
	}
	// TODO: normalize unicode names, in order to be searchable and collapse dupes
	printableTagNames := removeUnprintableCharacters(rawTagNames)
	trimmedTagNames := trimTagNames(printableTagNames)
	nonEmptyTagNames := removeEmptyTagNames(trimmedTagNames)
	uniqueTagNames := removeDuplicateTagNames(nonEmptyTagNames)

	return uniqueTagNames, nil
}
