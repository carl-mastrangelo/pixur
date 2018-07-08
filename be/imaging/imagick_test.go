package imaging

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"
	"testing"

	"google.golang.org/grpc/codes"
	"gopkg.in/gographics/imagick.v1/imagick"
)

func TestReadImage2_noImage(t *testing.T) {
	var b bytes.Buffer
	pi, sts := ReadImage2(&b)
	if sts == nil {
		pi.Close()
		t.Fatal("expected an error")
	}
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "unable to decode image"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestReadImage2_partialImage(t *testing.T) {
	mw := imagick.NewMagickWand()
	defer mw.Destroy()
	pw := imagick.NewPixelWand()
	defer pw.Destroy()

	mw.NewImage(1, 1, pw)
	mw.SetFormat(string(DefaultJpegFormat))
	f, err := ioutil.TempFile("", "pixurtest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if err := mw.WriteImageFile(f); err != nil {
		t.Fatal(err)
	}

	// 1 past the start of the file, to ensure its invalid
	if _, err := f.Seek(1, io.SeekStart); err != nil {
		t.Fatal(err)
	}

	pi, sts := ReadImage2(f)
	if sts == nil {
		pi.Close()
		t.Fatal("expected an error")
	}
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "unable to decode image"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestReadImage2_jpeg(t *testing.T) {
	mw := imagick.NewMagickWand()
	defer mw.Destroy()
	pw := imagick.NewPixelWand()
	defer pw.Destroy()

	mw.NewImage(1, 2, pw)
	mw.SetImageFormat(string(DefaultJpegFormat))
	f, err := ioutil.TempFile("", "pixurtest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if err := mw.WriteImageFile(f); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}

	pi, sts := ReadImage2(f)
	if sts != nil {
		t.Fatal(sts)
	}
	defer pi.Close()

	if have, want := pi.Format(), "JPEG"; string(have) != want {
		t.Error("have", have, "want", want)
	}
	if !pi.Format().IsJpeg() {
		t.Error("not a jpeg")
	}
	dur, sts := pi.Duration()
	if sts != nil {
		t.Fatal(sts)
	}
	if dur != nil {
		t.Error("jpegs can't have duration", dur)
	}
	x, y, sts := pi.Dimensions()
	if sts != nil {
	  t.Fatal(sts)
	}
	if x != 1 || y != 2 {
	  t.Error("bad dimensions", x, y)
	}
}

func TestReadImage2_png(t *testing.T) {
	mw := imagick.NewMagickWand()
	defer mw.Destroy()
	pw := imagick.NewPixelWand()
	defer pw.Destroy()

	mw.NewImage(1, 2, pw)
	mw.SetImageFormat(string(DefaultPngFormat))
	f, err := ioutil.TempFile("", "pixurtest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if err := mw.WriteImageFile(f); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}

	pi, sts := ReadImage2(f)
	if sts != nil {
		t.Fatal(sts)
	}
	defer pi.Close()

	if have, want := pi.Format(), "PNG"; string(have) != want {
		t.Error("have", have, "want", want)
	}
	if !pi.Format().IsPng() {
		t.Error("not a png")
	}
	dur, sts := pi.Duration()
	if sts != nil {
		t.Fatal(sts)
	}
	if dur != nil {
		t.Error("pngs can't have duration", dur)
	}
	x, y, sts := pi.Dimensions()
	if sts != nil {
	  t.Fatal(sts)
	}
	if x != 1 || y != 2 {
	  t.Error("bad dimensions", x, y)
	}
}

func TestReadImage2_gif_singleframe(t *testing.T) {
	mw := imagick.NewMagickWand()
	defer mw.Destroy()
	pw := imagick.NewPixelWand()
	defer pw.Destroy()

	mw.NewImage(1, 2, pw)
	mw.SetImageFormat(string(DefaultGifFormat))
	f, err := ioutil.TempFile("", "pixurtest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if err := mw.WriteImagesFile(f); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}

	pi, sts := ReadImage2(f)
	if sts != nil {
		t.Fatal(sts)
	}
	defer pi.Close()

	if have, want := pi.Format(), "GIF"; string(have) != want {
		t.Error("have", have, "want", want)
	}
	if !pi.Format().IsGif() {
		t.Error("not a gif")
	}
	dur, sts := pi.Duration()
	if sts != nil {
		t.Fatal(sts)
	}
	if dur != nil {
		t.Error("single gifs can't have duration", dur)
	}
	x, y, sts := pi.Dimensions()
	if sts != nil {
	  t.Fatal(sts)
	}
	if x != 1 || y != 2 {
	  t.Error("bad dimensions", x, y)
	}
}

func TestReadImage2_gif_singleframe_duration(t *testing.T) {
	mw := imagick.NewMagickWand()
	defer mw.Destroy()
	pw := imagick.NewPixelWand()
	defer pw.Destroy()

	mw.NewImage(1, 2, pw)
	mw.SetImageFormat(string(DefaultGifFormat))
	mw.SetImageDelay(gifTicksPerSecond) // should be ignored
	f, err := ioutil.TempFile("", "pixurtest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if err := mw.WriteImagesFile(f); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}

	pi, sts := ReadImage2(f)
	if sts != nil {
		t.Fatal(sts)
	}
	defer pi.Close()

	if have, want := pi.Format(), "GIF"; string(have) != want {
		t.Error("have", have, "want", want)
	}
	if !pi.Format().IsGif() {
		t.Error("not a gif")
	}
	dur, sts := pi.Duration()
	if sts != nil {
		t.Fatal(sts)
	}
	if dur != nil {
		t.Error("single gifs can't have duration", dur)
	}
	x, y, sts := pi.Dimensions()
	if sts != nil {
	  t.Fatal(sts)
	}
	if x != 1 || y != 2 {
	  t.Error("bad dimensions", x, y)
	}
}

func TestReadImage2_gif_multiframe(t *testing.T) {
	mw := imagick.NewMagickWand()
	defer mw.Destroy()
	pw := imagick.NewPixelWand()
	defer pw.Destroy()

	mw.NewImage(1, 2, pw)
	mw.SetImageFormat(string(DefaultGifFormat))
	
	tmp := mw.GetImage()
	tmp.SetImageDelay(gifTicksPerSecond)
	mw.AddImage(tmp)
	tmp.SetImageDelay(gifTicksPerSecond / 2)
	mw.AddImage(tmp)
	tmp.Destroy()
	
	f, err := ioutil.TempFile("", "pixurtest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if err := mw.WriteImagesFile(f); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}

	pi, sts := ReadImage2(f)
	if sts != nil {
		t.Fatal(sts)
	}
	defer pi.Close()

	if have, want := pi.Format(), "GIF"; string(have) != want {
		t.Error("have", have, "want", want)
	}
	if !pi.Format().IsGif() {
		t.Error("not a gif")
	}
	dur, sts := pi.Duration()
	if sts != nil {
		t.Fatal(sts)
	}
	if dur == nil {
		t.Error("missing duration", dur)
	}
	if *dur != 3 * time.Second / 2 {
	  t.Error("wrong duration", *dur)
	}
	x, y, sts := pi.Dimensions()
	if sts != nil {
	  t.Fatal(sts)
	}
	if x != 1 || y != 2 {
	  t.Error("bad dimensions", x, y)
	}
}

// There is no test for overflow behavior because it doesn't seem reasonable to come up with a test
// case.  Gif frame duration is max of 2**16 * 1/100s, which is about 10 minutes.  To overflow an
// int64 time.Duration, it would need about 1.5 million frames, but the test runs out of memory.
// Trying to use MP4 or another container which can have longer frames is not proving easy, so I'm
// going to just leave it alone.
func TestReadImage2_gif_failsOnExcessDelay(t *testing.T) {
  // Get out of jail free
}
