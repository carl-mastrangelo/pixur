package tasks

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"image"
	"image/color"
	"image/gif"
	"io"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"pixur.org/pixur/schema"
	ptest "pixur.org/pixur/testing"
)

type container struct {
	t       testing.TB
	db      *sql.DB
	tempDir string
}

func NewContainer(t testing.TB) *container {
	if t == nil {
		panic("t cannot be nil")
	}
	return &container{
		t: t,
	}
}

func (c *container) GetDB() *sql.DB {
	if c.db == nil {
		db, err := ptest.GetDB()
		if err != nil {
			c.t.Fatal(err)
		}
		if err := schema.CreateTables(db); err != nil {
			c.t.Fatal(err)
		}
		c.db = db
	}
	return c.db
}

func (c *container) GetTempDir() string {
	if c.tempDir != "" {
		return c.tempDir
	}
	if path, err := ioutil.TempDir("", "unitTestTempDir"); err != nil {
		c.t.Fatal(err)
	} else {
		c.tempDir = path
	}
	return c.tempDir
}

func (c *container) CleanUp() {
	if c.db != nil {
		if err := c.db.Close(); err != nil {
			c.t.Error(err)
		}
	}
	c.db = nil
	if c.tempDir != "" {
		if err := os.RemoveAll(c.tempDir); err != nil {
			c.t.Error(err)
		}
	}
	c.tempDir = ""
}

func (c *container) CreatePic() *schema.Pic {
	h1 := sha256.New()
	h2 := sha1.New()
	h3 := md5.New()
	if err := binary.Write(io.MultiWriter(h1, h2, h3), binary.LittleEndian, rand.Int63()); err != nil {
		c.t.Fatal(err)
	}
	p := &schema.Pic{}

	if err := p.Insert(c.GetDB()); err != nil {
		c.t.Fatal(err, p)
	}

	pi1 := &schema.PicIdentifier{
		PicId: p.PicId,
		Type:  schema.PicIdentifier_SHA256,
		Value: h1.Sum(nil),
	}
	if err := pi1.Insert(c.GetDB()); err != nil {
		c.t.Fatal(err, p)
	}

	pi2 := &schema.PicIdentifier{
		PicId: p.PicId,
		Type:  schema.PicIdentifier_SHA1,
		Value: h2.Sum(nil),
	}
	if err := pi2.Insert(c.GetDB()); err != nil {
		c.t.Fatal(err, p)
	}

	pi3 := &schema.PicIdentifier{
		PicId: p.PicId,
		Type:  schema.PicIdentifier_MD5,
		Value: h3.Sum(nil),
	}
	if err := pi3.Insert(c.GetDB()); err != nil {
		c.t.Fatal(err, p)
	}

	if err := c.writeImageData(p); err != nil {
		c.t.Fatal(err)
	}
	if err := c.writeThumbnailData(p); err != nil {
		c.t.Fatal(err)
	}

	return p
}

func (c *container) CreateTag() *schema.Tag {
	dictionary := "abcdefghijklmnopqrstuvwxyz"
	var name string
	for i := 0; i < 6; i++ {
		name += string(dictionary[rand.Intn(len(dictionary))])
	}
	t := &schema.Tag{Name: name}
	if err := t.Insert(c.GetDB()); err != nil {
		c.t.Fatal(err)
	}

	return t
}

func (c *container) CreatePicTag(p *schema.Pic, t *schema.Tag) *schema.PicTag {
	picTag := &schema.PicTag{
		PicId: p.PicId,
		TagId: t.TagId,
		Name:  t.Name,
	}
	if _, err := picTag.Insert(c.GetDB()); err != nil {
		c.t.Fatal(err)
	}
	t.UsageCount++
	if err := t.Update(c.GetDB()); err != nil {
		c.t.Fatal(err)
	}

	return picTag
}

func (c *container) RefreshPic(p **schema.Pic) {
	stmt, err := schema.PicPrepare("SELECT * FROM_ WHERE %s = ?;", c.GetDB(), schema.PicColId)
	if err != nil {
		c.t.Fatal(err)
	}
	updated, err := schema.LookupPic(stmt, (*p).PicId)
	if err == sql.ErrNoRows {
		*p = nil
	} else if err != nil {
		c.t.Fatal(err)
	}
	*p = updated
}

func (c *container) UpdatePic(p *schema.Pic) {
	if err := p.Update(c.GetDB()); err != nil {
		c.t.Fatal(err)
	}
}

func (c *container) RefreshTag(t **schema.Tag) {
	stmt, err := schema.TagPrepare("SELECT * FROM_ WHERE %s = ?;", c.GetDB(), schema.TagColId)
	if err != nil {
		c.t.Fatal(err)
	}
	updated, err := schema.LookupTag(stmt, (*t).TagId)
	if err == sql.ErrNoRows {
		*t = nil
	} else if err != nil {
		c.t.Fatal(err)
	}
	*t = updated
}

func (c *container) RefreshPicTag(pt **schema.PicTag) {
	stmt, err := schema.PicTagPrepare("SELECT * FROM_ WHERE %s = ? AND %s = ?;",
		c.GetDB(), schema.PicTagColPicId, schema.PicTagColTagId)
	if err != nil {
		c.t.Fatal(err)
	}
	updated, err := schema.LookupPicTag(stmt, (*pt).PicId, (*pt).TagId)
	if err == sql.ErrNoRows {
		*pt = nil
	} else if err != nil {
		c.t.Fatal(err)
	}
	*pt = updated
}

func (c *container) getRandomImageData() *bytes.Reader {
	bounds := image.Rect(0, 0, 5, 10)
	img := image.NewGray(bounds)
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			img.SetGray(x, y, color.Gray{Y: uint8(rand.Int31n(math.MaxUint8))})
		}
	}
	f := bytes.NewBuffer(nil)

	if err := gif.Encode(f, img, &gif.Options{}); err != nil {
		c.t.Fatal(err)
	}
	return bytes.NewReader(f.Bytes())
}

func (c *container) writeImageData(p *schema.Pic) error {
	path := p.Path(c.GetTempDir())
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0770); err != nil {
		c.t.Fatal(err)
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, c.getRandomImageData()); err != nil {
		return err
	}
	return nil
}

func (c *container) writeThumbnailData(p *schema.Pic) error {
	path := p.ThumbnailPath(c.GetTempDir())
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0770); err != nil {
		c.t.Fatal(err)
	}
	f, err := os.Create(p.ThumbnailPath(c.GetTempDir()))
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, c.getRandomImageData()); err != nil {
		return err
	}
	return nil
}

func runTests(m *testing.M) int {
	defer ptest.CleanUp()

	return m.Run()
}

func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}
