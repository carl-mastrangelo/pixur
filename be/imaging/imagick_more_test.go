package imaging

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/gographics/imagick.v1/imagick"
)

func TestEvenMore(t *testing.T) {
	f, err := os.Open("/tmp/better.png")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	im, sts := ReadImage(f)
	if sts != nil {
		t.Fatal(sts)
	}
	defer im.Close()
	t1, sts := im.Thumbnail()
	if sts != nil {
		panic(sts)
	}
	defer t1.Close()
	out, err := os.Create("/tmp/RR.jpg")
	if err != nil {
		panic(err)
	}
	defer out.Close()
	t1.Write(out)
}

func TesvtReadAll(t *testing.T) {
	cc := make(map[ImageFormat]int)
	cc2 := make(map[imagick.ColorspaceType]int)
	cc3 := make(map[string][]string)
	cc4 := make(map[string][]string)

	i := 0
	bad := filepath.Walk("/home/carl/go/src/pixur.org/pixur/pix/", func(path string, fi os.FileInfo, e error) error {
		if e != nil {
			return e
		}
		if i > 50000 {
			return nil
		}
		i++
		if fi.IsDir() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		prefix := make([]byte, 4)
		if n, err := f.ReadAt(prefix, 0); err != nil || n != 4 {
			t.Fatal(err, n)
		}
		if string(prefix) == ebmlHeader {
			return nil
		}
		pi, sts := ReadImage(f)
		if sts != nil {
			return sts
		}
		defer pi.Close()

		ri, sts := ReadImageOld(io.NewSectionReader(f, 0, 100000000))
		if sts != nil {
			println(f.Name())
			t.Fatal(sts)
		}

		bb, bbf := PerceptualHash0(ri)

		_ = bbf

		hash, grays, sts := pi.PerceptualHash0()
		if sts != nil {
			t.Fatal(sts)
		}

		_ = grays
		bitty := CountBits(binary.LittleEndian.Uint64(bb) ^ binary.LittleEndian.Uint64(hash))
		if bitty >= 7 {
			fmt.Printf("%02x\n", hash)
			fmt.Printf("%02x\n", bb)
			println("welp", bitty, path)
		}

		if true {
			return nil
		}
		/*
			pi2, sts := pi.Thumbnail()
			if sts != nil {
				t.Log(path)
				return sts
			}
			defer pi2.Close()

			ouut, err := os.Create("/tmp/" + filepath.Base(path) + "." + string(pi2.Format()))
			if err != nil {
				return err
			}
			defer ouut.Close()
			if sts := pi2.Write(ouut); sts != nil {
				return sts
			}
			println("wrote" + ouut.Name())
		*/

		return nil
	})
	if bad != nil {
		t.Error(bad)
	}

	fmt.Println(cc)
	fmt.Println(cc2)
	fmt.Println("OPTS:")
	for k, c := range cc3 {
		println(k, len(c), c[0])
	}
	fmt.Println("ARTIFACTS:")
	for k, c := range cc4 {
		println(k, len(c), c[0])
	}
}
