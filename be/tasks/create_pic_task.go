package tasks

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"image"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"pixur.org/pixur/be/imaging"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

type readAtSeeker interface {
	io.Reader
	io.ReaderAt
	io.Seeker
}

type CreatePicTask struct {
	// Deps
	PixPath string
	DB      db.DB

	// Inputs
	Filename string
	FileData readAtSeeker
	TagNames []string

	// Alternatively, a url can be uploaded
	FileURL string

	// State
	// The file that was created to hold the upload.
	tempFilename string
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
}

func (t *CreatePicTask) Run(ctx context.Context) (sCap status.S) {
	var err error
	var sts status.S
	t.now = time.Now()
	wf, err := ioutil.TempFile(t.PixPath, "__")
	if err != nil {
		return status.InternalError(err, "Unable to create tempfile")
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
		p.Source = []*schema.Pic_FileSource{{
			Url:       t.FileURL,
			CreatedTs: schema.ToTspb(t.now),
		}}
	} else {
		return status.InvalidArgument(nil, "No file uploaded")
	}
	if t.Filename != "" {
		p.FileName = []string{t.Filename}
	}

	img, err := imaging.FillImageConfig(wf, p)
	if err != nil {
		if err, ok := err.(*imaging.BadWebmFormatErr); ok {
			return status.InvalidArgument(err, "Bad Web Fmt")
		}
		return status.InvalidArgument(err, "Bad Image")
	}

	thumbnail := imaging.MakeThumbnail(img)

	j, err := tab.NewJob(ctx, t.DB)
	if err != nil {
		return status.InternalError(err, "can't create job")
	}
	defer cleanUp(j, &sCap)

	u, sts := requireCapability(ctx, j, schema.User_PIC_CREATE)
	if sts != nil {
		return sts
	}

	identities, sts := generatePicIdentities(wf)
	if sts != nil {
		return sts
	}

	sha256Type := schema.PicIdent_SHA256
	sha256Value := identities[sha256Type]
	pis, err := j.FindPicIdents(db.Opts{
		Prefix: tab.PicIdentsIdent{
			Type:  &sha256Type,
			Value: &sha256Value,
		},
		Lock:  db.LockWrite,
		Limit: 1,
	})
	if err != nil {
		return status.InternalError(err, "can't find pic idents")
	}
	if len(pis) != 0 {
		return status.AlreadyExists(nil, "pic already exists")
	}

	picID, err := j.AllocID()
	if err != nil {
		status.InternalError(err, "can't allocate id")
	}
	p.PicId = picID

	if err := j.InsertPic(p); err != nil {
		return status.InternalError(err, "can't insert pic")
	}

	if err := t.renameTempFile(p); err != nil {
		return err
	}

	// If there is a problem creating the thumbnail, just continue on.
	if err := imaging.SaveThumbnail(thumbnail, p, t.PixPath); err != nil {
		log.Println("WARN Failed to create thumbnail", err)
	}

	tags, sts := t.insertOrFindTags(j)
	if sts != nil {
		return sts
	}
	// must happen after pic is created, because it depends on pic id
	if sts := t.addTagsForPic(p, tags, u.UserId, j); sts != nil {
		return sts
	}

	// This also must happen after the pic is inserted, to use PicId
	for typ, val := range identities {
		ident := &schema.PicIdent{
			PicId: p.PicId,
			Type:  typ,
			Value: val,
		}
		if err := j.InsertPicIdent(ident); err != nil {
			return status.InternalError(err, "can't create pic ident")
		}
	}

	pIdent := getPerceptualHash(p, img)
	if err := j.InsertPicIdent(pIdent); err != nil {
		return status.InternalError(err, "can't create pic ident")
	}

	if err := j.Commit(); err != nil {
		return status.InternalError(err, "can't commit job")
	}

	// The upload succeeded
	t.tempFilename = ""
	t.CreatedPic = p
	return nil
}

// Moves the uploaded file and records the file size.  It might not be possible to just move the
// file in the event that the uploaded location is on a different partition than persistent dir.
func (t *CreatePicTask) moveUploadedFile(tempFile io.Writer, p *schema.Pic) status.S {
	// If the task is reset, this will need to seek to the beginning
	if _, err := t.FileData.Seek(0, os.SEEK_SET); err != nil {
		return status.InternalError(err, "Can't Seek")
	}
	// TODO: check if the t.FileData is an os.File, and then try moving it.
	if bytesWritten, err := io.Copy(tempFile, t.FileData); err != nil {
		return status.InternalError(err, "Unable to move uploaded file")
	} else {
		p.FileSize = bytesWritten
	}
	// Attempt to flush the file incase an outside program needs to read from it.
	if f, ok := tempFile.(*os.File); ok {
		if err := f.Sync(); err != nil {
			return status.InternalError(err, "Failed to sync uploaded file")
		}
	}
	return nil
}

func (t *CreatePicTask) downloadFile(tempFile io.Writer, p *schema.Pic) status.S {
	resp, err := http.Get(t.FileURL)
	if err != nil {
		return status.InvalidArgument(err, "Unable to download", t.FileURL)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return status.InvalidArgumentf(nil, "Failed to Download Pic %s [%d]",
			t.FileURL, resp.StatusCode)
	}

	if bytesWritten, err := io.Copy(tempFile, resp.Body); err != nil {
		return status.InternalError(err, "Failed to copy downloaded file")
	} else {
		p.FileSize = bytesWritten
	}
	// Attempt to flush the file incase an outside program needs to read from it.
	if f, ok := tempFile.(*os.File); ok {
		if err := f.Sync(); err != nil {
			return status.InternalError(err, "Failed to sync file")
		}
	}
	return nil
}

func (t *CreatePicTask) renameTempFile(p *schema.Pic) status.S {
	if err := os.MkdirAll(filepath.Dir(p.Path(t.PixPath)), 0770); err != nil {
		return status.InternalError(err, "Unable to prepare pic dir")
	}

	if err := os.Rename(t.tempFilename, p.Path(t.PixPath)); err != nil {
		return status.InternalErrorf(err, "Unable to move uploaded file %s -> %s",
			t.tempFilename, p.Path(t.PixPath))
	}
	// point this at the new file, incase the overall transaction fails
	t.tempFilename = p.Path(t.PixPath)
	return nil
}

// This function is not really transactional, because it hits multiple entity roots.
// TODO: test this.
func (t *CreatePicTask) insertOrFindTags(j *tab.Job) ([]*schema.Tag, status.S) {
	cleanedTags, sts := cleanTagNames(t.TagNames)
	if sts != nil {
		return nil, sts
	}
	sort.Strings(cleanedTags)
	var allTags []*schema.Tag
	for _, tagName := range cleanedTags {
		tag, sts := findAndUpsertTag(tagName, t.now, j)
		if sts != nil {
			return nil, sts
		}
		allTags = append(allTags, tag)
	}

	return allTags, nil
}

// findAndUpsertTag looks for an existing tag by name.  If it finds it, it updates the modified
// time and usage counter.  Otherwise, it creates a new tag with an initial count of 1.
func findAndUpsertTag(tagName string, now time.Time, j *tab.Job) (*schema.Tag, status.S) {
	tag, sts := findTagByName(tagName, j)
	if sts != nil {
		return nil, sts
	}
	if tag == nil {
		return createTag(tagName, now, j)
	}

	tag.SetModifiedTime(now)
	tag.UsageCount += 1
	if err := j.UpdateTag(tag); err != nil {
		return nil, status.InternalError(err, "can't update tag")
	}
	return tag, nil
}

func createTag(tagName string, now time.Time, j *tab.Job) (*schema.Tag, status.S) {
	id, err := j.AllocID()
	if err != nil {
		return nil, status.InternalError(err, "can't allocate id")
	}
	tag := &schema.Tag{
		TagId:      id,
		Name:       tagName,
		UsageCount: 1,
	}
	tag.SetCreatedTime(now)
	tag.SetModifiedTime(now)
	if err := j.InsertTag(tag); err != nil {
		return nil, status.InternalError(err, "can't create tag")
	}

	return tag, nil
}

func findTagByName(tagName string, j *tab.Job) (*schema.Tag, status.S) {
	tags, err := j.FindTags(db.Opts{
		Prefix: tab.TagsName{&tagName},
		Lock:   db.LockWrite,
		Limit:  1,
	})
	if err != nil {
		return nil, status.InternalError(err, "can't find tags")
	}
	if len(tags) != 1 {
		return nil, nil
	}
	return tags[0], nil
}

func (t *CreatePicTask) addTagsForPic(
	p *schema.Pic, tags []*schema.Tag, userID int64, j *tab.Job) status.S {
	for _, tag := range tags {
		picTag := &schema.PicTag{
			PicId:  p.PicId,
			TagId:  tag.TagId,
			Name:   tag.Name,
			UserId: userID,
		}
		picTag.SetCreatedTime(p.GetCreatedTime())
		picTag.SetModifiedTime(p.GetModifiedTime())
		if err := j.InsertPicTag(picTag); err != nil {
			return status.InternalError(err, "can't create pic tag")
		}
	}
	return nil
}

func checkValidUnicode(tagNames []string) status.S {
	for _, tn := range tagNames {
		if !utf8.ValidString(tn) {
			return status.InvalidArgument(nil, "Invalid tag name", tn)
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

func cleanTagNames(rawTagNames []string) ([]string, status.S) {
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

func getPerceptualHash(p *schema.Pic, im image.Image) *schema.PicIdent {
	hash, inputs := imaging.PerceptualHash0(im)
	return &schema.PicIdent{
		PicId:      p.PicId,
		Type:       schema.PicIdent_DCT_0,
		Value:      hash,
		Dct0Values: inputs,
	}
}

func generatePicIdentities(f io.ReadSeeker) (map[schema.PicIdent_Type][]byte, status.S) {
	if _, err := f.Seek(0, os.SEEK_SET); err != nil {
		return nil, status.InternalError(err, "Can't Seek")
	}
	defer f.Seek(0, os.SEEK_SET)
	h1 := sha256.New()
	h2 := sha1.New()
	h3 := md5.New()

	w := io.MultiWriter(h1, h2, h3)

	if _, err := io.Copy(w, f); err != nil {
		return nil, status.InternalError(err, "Can't Copy")
	}
	return map[schema.PicIdent_Type][]byte{
		schema.PicIdent_SHA256: h1.Sum(nil),
		schema.PicIdent_SHA1:   h2.Sum(nil),
		schema.PicIdent_MD5:    h3.Sum(nil),
	}, nil
}
