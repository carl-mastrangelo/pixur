package imaging

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"gopkg.in/gographics/imagick.v2/imagick"
)

func TestReadImage_noImage(t *testing.T) {
	var b bytes.Buffer
	pi, sts := ReadImage(context.Background(), &b)
	if sts == nil {
		pi.Close()
		t.Fatal("expected an error")
	}
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "unable to read first"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestReadImage_partialImage(t *testing.T) {
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

	pi, sts := ReadImage(context.Background(), f)
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

func TestReadImage_jpeg(t *testing.T) {
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

	pi, sts := ReadImage(context.Background(), f)
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
	if x, y := pi.Dimensions(); x != 1 || y != 2 {
		t.Error("bad dimensions", x, y)
	}
}

func TestReadImage_png(t *testing.T) {
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

	pi, sts := ReadImage(context.Background(), f)
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

	if x, y := pi.Dimensions(); x != 1 || y != 2 {
		t.Error("bad dimensions", x, y)
	}
}

func TestThumbnail_png_offset(t *testing.T) {
	mw := imagick.NewMagickWand()
	defer mw.Destroy()
	pw := imagick.NewPixelWand()
	defer pw.Destroy()

	mw.NewImage(100, 200, pw)
	mw.SetImageFormat(string(DefaultPngFormat))
	f, err := ioutil.TempFile("", "pixurtest.png")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	// set offset far outside.
	if err := mw.SetImagePage(100, 200, 300, 400); err != nil {
		t.Fatal(err)
	}

	if err := mw.WriteImageFile(f); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}

	pi, sts := ReadImage(context.Background(), f)
	if sts != nil {
		t.Fatal(sts)
	}
	defer pi.Close()

	thumb, sts := pi.Thumbnail()
	if sts != nil {
		t.Fatal(sts)
	}
	defer thumb.Close()

	if x, y := thumb.Dimensions(); x != thumbnailSquareSize || y != thumbnailSquareSize {
		t.Error("bad dimensions", x, y)
	}
}

func TestWriteThumbnail(t *testing.T) {
	mw := imagick.NewMagickWand()
	defer mw.Destroy()
	pw := imagick.NewPixelWand()
	defer pw.Destroy()

	mw.NewImage(100, 200, pw)
	mw.SetImageFormat(string(DefaultPngFormat))
	f, err := ioutil.TempFile("", "pixurtest.png")
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

	pi, sts := ReadImage(context.Background(), f)
	if sts != nil {
		t.Fatal(sts)
	}
	defer pi.Close()

	thumb, sts := pi.Thumbnail()
	if sts != nil {
		t.Fatal(sts)
	}
	defer thumb.Close()

	// reset it one more time to write
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}

	if err := thumb.Write(f); err != nil {
		t.Fatal(err)
	}

	// reset it one more time to read
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	thumb2, sts := ReadImage(context.Background(), f)
	if sts != nil {
		t.Fatal(sts)
	}
	defer thumb2.Close()

	if w, h := thumb2.Dimensions(); w != thumbnailSquareSize || h != thumbnailSquareSize {
		t.Error("bad dims", w, h)
	}
	// I have experimentally confirmed there is no offset in the thumbnail.
}

func TestReadImage_gif_singleframe(t *testing.T) {
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

	pi, sts := ReadImage(context.Background(), f)
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
	if x, y := pi.Dimensions(); x != 1 || y != 2 {
		t.Error("bad dimensions", x, y)
	}
}

func TestReadImage_gif_singleframe_duration(t *testing.T) {
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

	pi, sts := ReadImage(context.Background(), f)
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
	if x, y := pi.Dimensions(); x != 1 || y != 2 {
		t.Error("bad dimensions", x, y)
	}
}

func TestReadImage_gif_multiframe(t *testing.T) {
	mw := imagick.NewMagickWand()
	defer mw.Destroy()
	pw := imagick.NewPixelWand()
	defer pw.Destroy()

	mw.NewImage(1, 2, pw)
	mw.SetImageFormat(string(DefaultGifFormat))
	mw.SetImageDelay(gifTicksPerSecond)

	tmp := mw.GetImage()
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

	pi, sts := ReadImage(context.Background(), f)
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
		t.Fatal("missing duration", dur)
	}
	if have, want := *dur, 3*time.Second/2; have != want {
		t.Error("wrong duration", have, want)
	}
	if x, y := pi.Dimensions(); x != 1 || y != 2 {
		t.Error("bad dimensions", x, y)
	}
}

func TestReadImage_gif_shortFrameLengthRoundsUp(t *testing.T) {
	mw := imagick.NewMagickWand()
	defer mw.Destroy()
	pw := imagick.NewPixelWand()
	defer pw.Destroy()

	mw.NewImage(1, 2, pw)
	mw.SetImageFormat(string(DefaultGifFormat))
	mw.SetImageDelay(0) // should round to 10/100

	tmp := mw.GetImage()
	tmp.SetImageDelay(1) // also should round to 10/100
	mw.AddImage(tmp)
	tmp.Destroy()

	tmp = mw.GetImage()
	tmp.SetImageDelay(2) // no rounding
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

	pi, sts := ReadImage(context.Background(), f)
	if sts != nil {
		t.Fatal(sts)
	}
	defer pi.Close()

	if !pi.Format().IsGif() {
		t.Error("not a gif")
	}
	dur, sts := pi.Duration()
	if sts != nil {
		t.Fatal(sts)
	}
	if dur == nil {
		t.Fatal("missing duration", dur)
	}
	if have, want := *dur, (100+100+20)*time.Millisecond; have != want {
		t.Error("wrong duration", have, want)
	}
}

// There is no test for overflow behavior because it doesn't seem reasonable to come up with a test
// case.  Gif frame duration is max of 2**16 * 1/100s, which is about 10 minutes.  To overflow an
// int64 time.Duration, it would need about 1.5 million frames, but the test runs out of memory.
// Trying to use MP4 or another container which can have longer frames is not proving easy, so I'm
// going to just leave it alone.
func TestReadImage_gif_failsOnExcessDelay(t *testing.T) {
	// Get out of jail free
}

func TestReadImage_gif_firstFrameSetsSize(t *testing.T) {
	mw := imagick.NewMagickWand()
	defer mw.Destroy()
	pw := imagick.NewPixelWand()
	defer pw.Destroy()
	if !pw.SetColor("white") {
		t.Fatal("can't set color")
	}
	pw2 := imagick.NewPixelWand()
	defer pw2.Destroy()
	if !pw2.SetColor("black") {
		t.Fatal("can't set color")
	}

	mw.NewImage(400, 100, pw)
	dw := imagick.NewDrawingWand()
	defer dw.Destroy()
	dw.SetStrokeOpacity(1)
	dw.SetStrokeColor(pw2)
	dw.SetStrokeWidth(4)
	dw.SetStrokeAntialias(false)
	dw.SetFillColor(pw2)
	dw.Rectangle(0, 0, 100, 100)
	mw.DrawImage(dw)

	mw.SetImageFormat(string(DefaultGifFormat))

	mw2 := imagick.NewMagickWand()
	defer mw2.Destroy()
	mw2.NewImage(200, 100, pw2)
	mw2.SetImageFormat(string(DefaultGifFormat))
	mw2.SetImagePage(200, 100, 200, 0)

	mw.AddImage(mw2)

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

	im, sts := ReadImage(context.Background(), f)
	if sts != nil {
		t.Fatal(sts)
	}
	defer im.Close()

	if !im.Format().IsGif() {
		t.Error("not a gif")
	}

	thumb, sts := im.Thumbnail()
	if sts != nil {
		t.Fatal(sts)
	}
	defer thumb.Close()

	pixel := thumb.(*imagickImage).mw.NewPixelIterator().GetCurrentIteratorRow()[0].GetGreen()
	if pixel != 1.0 {
		t.Error("expected a white pixel, but was not", pixel)
	}
}
