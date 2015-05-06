package pixur

import (
	"bytes"
	"database/sql"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"io"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"testing"

	"pixur.org/pixur/schema"
	ptest "pixur.org/pixur/testing"
)

type container struct {
	t  *testing.T
	db *sql.DB

	pixPath       string
	createdPicIds []schema.PicId
	createdTagIds []schema.TagId
}

func (c *container) CreatePic() *schema.Pic {
	p := &schema.Pic{}
	if err := p.InsertAndSetId(c.db); err != nil {
		c.t.Fatal(err)
	}
	c.createdPicIds = append(c.createdPicIds, p.Id)
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
	if err := t.InsertAndSetId(c.db); err != nil {
		c.t.Fatal(err)
	}
	c.createdTagIds = append(c.createdTagIds, t.Id)

	return t
}

func (c *container) CleanUp() {
	for _, picId := range c.createdPicIds {
		if _, err := (&schema.Pic{Id: picId}).Delete(c.db); err != nil {
			c.t.Error(err)
		}
	}
	for _, tagId := range c.createdTagIds {
		if _, err := (&schema.Tag{Id: tagId}).Delete(c.db); err != nil {
			c.t.Error(err)
		}
	}
	if c.pixPath != "" {
		if err := os.RemoveAll(c.pixPath); err != nil {
			c.t.Error(err)
		}
	}
}

func (c *container) mkPixPath() string {
	if c.pixPath != "" {
		return c.pixPath
	}
	if path, err := ioutil.TempDir("", "unitTestPixPath"); err != nil {
		c.t.Fatal(err)
	} else {
		c.pixPath = path
	}
	return c.pixPath
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
	f, err := os.Create(p.Path(c.mkPixPath()))
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
	f, err := os.Create(p.ThumbnailPath(c.mkPixPath()))
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, c.getRandomImageData()); err != nil {
		return err
	}
	return nil
}

var (
	testDB         *sql.DB
	_testSetups    []func() error
	_testTearDowns []func() error
)

func BeforeTestSuite(before func() error) {
	_testSetups = append(_testSetups, before)
}

func AfterTestSuite(after func() error) {
	_testTearDowns = append(_testTearDowns, after)
}

func init() {
	BeforeTestSuite(func() error {
		db, err := ptest.GetDB()
		if err != nil {
			return err
		}
		AfterTestSuite(func() error {
			ptest.CleanUp()
			return nil
		})
		testDB = db
		if err := createTables(db); err != nil {
			return err
		}
		return nil
	})
}

func runTests(m *testing.M) int {
	defer func() {
		for _, after := range _testTearDowns {
			if err := after(); err != nil {
				fmt.Println("Error in teardown", err)
			}
		}
	}()

	for _, before := range _testSetups {
		if err := before(); err != nil {
			fmt.Println("Error in test setup", err)
			return 1
		}
	}

	return m.Run()
}

func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}
