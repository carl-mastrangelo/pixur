package imaging

import (
	"bytes"
	"io"
	"math"
	"os"
	"time"

	// this is the only outside pixur package dependency.  Avoid depending too much on schema.
	"pixur.org/pixur/be/status"

	// unfortuantely, most of my machines are stuck on 6.8.x.x
	// TODO: update this to v2 when possible.
	"gopkg.in/gographics/imagick.v1/imagick"
)

const (
	// WEBM magic header
	ebmlHeader = "\x1a\x45\xdf\xa3"
)

const (
	DefaultGifFormat  ImageFormat = "GIF"
	DefaultJpegFormat ImageFormat = "JPEG"
	DefaultPngFormat  ImageFormat = "PNG"
)

const (
	thumbnailSquareSize = 160
)

const gifTicksPerSecond = 100

func init() {
	imagick.Initialize()
	// never call Terminate
}

// a list of all formats: mw.QueryFormats("*")

// ImageFormat is the string format of the image
type ImageFormat string

// IsGif returns true if the type of this image is a GIF.
func (f ImageFormat) IsGif() bool {
	return f == "GIF" || f == "GIF87"
}

// IsJpeg returns true if the type of this image is a JPEG.
func (f ImageFormat) IsJpeg() bool {
	return f == "JPE" || f == "JPEG" || f == "JPG"
}

// IsPng returns true if the type of this image is a PNG.
func (f ImageFormat) IsPng() bool {
	// These are the types imagick says it supports: PNG PNG00 PNG24 PNG32 PNG48 PNG64 PNG8.  I am
	// only including the ones I know.
	return f == "PNG" || f == "PNG24" || f == "PNG32" || f == "PNG8"
}

type PixurImage2 interface {
	Format() ImageFormat
	Dimensions() (width, height uint)
	// In the future, this could also include a histogram
	Duration() (*time.Duration, status.S)

	Thumbnail() (PixurImage2, status.S)

	Write(io.Writer) status.S

	Close()
}

var _ PixurImage2 = (*pixurImage2)(nil)

type pixurImage2 struct {
	mw *imagick.MagickWand
}

func (pi *pixurImage2) Write(w io.Writer) status.S {
	// TODO: maybe make this work with GIF?  I don't think there is a case that Pixur wants to write
	// back out a non-thumbnail.  All thumbnails are single image.
	switch w := w.(type) {
	case *os.File:
		if err := pi.mw.WriteImageFile(w); err != nil {
			return status.InternalError(err, "can't write image file")
		}
	default:
		pi.mw.ResetIterator()
		if _, err := w.Write(pi.mw.GetImageBlob()); err != nil {
			return status.InternalError(err, "can't write image")
		}
		panic("uh oh")
	}

	return nil
}

func (pi *pixurImage2) Thumbnail() (PixurImage2, status.S) {
	w, h := pi.Dimensions()
	var neww, newh uint
	var x, y int
	if w > h {
		neww = h
		newh = h
		x = int((w - neww) / 2)
		y = 0
	} else {
		neww = w
		newh = w
		x = 0
		y = int((h - newh) / 2)
	}
	newmw := pi.mw.Clone()
	newmw.ResetIterator()
	// Some PNGs have an `oFFs` section that messes with the crop
	if err := newmw.SetImagePage(w, h, 0, 0); err != nil {
		return nil, status.InternalError(err, "unable to repage image")
	}

	destroy := true
	defer func() {
		if destroy {
			newmw.Destroy()
		}
	}()

	if err := newmw.CropImage(neww, newh, x, y); err != nil {
		return nil, status.InternalError(err, "unable to crop thumbnail")
	}

	side := uint(thumbnailSquareSize)
	newmw.TransformImageColorspace(imagick.COLORSPACE_RGB)
	if err := newmw.ResizeImage(side, side, imagick.FILTER_CATROM, 1); err != nil {
		return nil, status.InternalError(err, "unable to resize thumbnail")
	}
	newmw.TransformImageColorspace(imagick.COLORSPACE_SRGB)

	for _, p := range newmw.GetImageProfiles("*") {
		switch p {
		case "icc":
			fallthrough // usually big, and we always convert to srgb
		default:
			newmw.RemoveImageProfile(p)
		}
	}

	for _, p := range newmw.GetImageProperties("*") {
		// I found these by looking through all properties of images I had and seeing which ones
		// ImageMagick reads (rather than informatively sets)
		switch p {
		case "png:bit-depth-written":
		case "png:IHDR.color-type-orig":
		case "png:IHDR.bit-depth-orig":

		case "jpeg:colorspace":
			fallthrough // we only want srgb
		case "jpeg:sampling-factor":
			fallthrough // we set this later
		case "date:modify":
			fallthrough // don't care
		case "png:tIME":
			fallthrough // don't care
		default:
			if err := newmw.DeleteImageProperty(p); err != nil {
				return nil, status.InternalError(err, "unable to delete property ", p)
			}
		}
	}

	// None of the images I found had options, but delete them anyways
	for _, o := range newmw.GetOptions("*") {
		if err := newmw.DeleteOption(o); err != nil {
			return nil, status.InternalError(err, "unable to delete option ", o)
		}
	}

	for _, a := range newmw.GetImageArtifacts("*") {
		if err := newmw.DeleteImageArtifact(a); err != nil {
			return nil, status.InternalError(err, "unable to delete artifact ", a)
		}
	}

	format := pi.Format()
	switch {
	case format.IsGif():
	case format.IsPng():
		if err := newmw.SetOption("png:exclude-chunks", "all"); err != nil {
			return nil, status.InternalError(err, "unable to exclude png chunks")
		}
		// TODO: maybe do this for JPEG?
		if err := newmw.SetOption("png:bit-depth", "8"); err != nil {
			return nil, status.InternalError(err, "unable to set png bit depth")
		}

	default:
		if err := newmw.SetImageFormat(string(DefaultJpegFormat)); err != nil {
			return nil, status.InternalError(err, "unable to set format")
		}
		fallthrough
	case format.IsJpeg():
		newmw.SetImageCompressionQuality(90)
		if err := newmw.SetOption("jpeg:sampling-factor", "1x1,1x1,1x1"); err != nil {
			return nil, status.InternalError(err, "unable to set sampling factor")
		}
	}

	// TODO:trim profiles (keep colorspace?), apply orientation,
	newpi := &pixurImage2{
		mw: newmw,
	}
	destroy = false
	return newpi, nil
}

func (pi *pixurImage2) Format() ImageFormat {
	return ImageFormat(pi.mw.GetImageFormat())
}

func (pi *pixurImage2) Dimensions() (w uint, h uint) {
	return pi.mw.GetImageWidth(), pi.mw.GetImageHeight()
}

func (pi *pixurImage2) Duration() (*time.Duration, status.S) {
	if !pi.Format().IsGif() {
		return nil, nil
	}
	if tps := pi.mw.GetImageTicksPerSecond(); tps != gifTicksPerSecond {
		return nil, status.InternalErrorf(nil, "Wrong ticks per second %v", tps)
	}
	switch pi.mw.GetNumberImages() {
	case 1:
		return nil, nil
	case 0:
		return nil, status.InvalidArgument(nil, "no images")
	}
	var d time.Duration
	pi.mw.ResetIterator()
	const tickDuration = time.Second / gifTicksPerSecond
	const delayTicksMax = math.MaxInt64 / int64(tickDuration)
	for {
		delayTicks := int64(pi.mw.GetImageDelay())
		if delayTicks < 0 || delayTicks > delayTicksMax {
			return nil, status.InvalidArgument(nil, "delayTicks would overflow", delayTicks)
		}
		// this should always be positive
		delayTickDuration := time.Duration(delayTicks) * tickDuration
		if d += delayTickDuration; d < 0 {
			return nil, status.InvalidArgument(nil, "duration overflow", d)
		}

		if !pi.mw.NextImage() {
			break
		}
	}

	return &d, nil
}

func (pi *pixurImage2) Close() {
	if pi.mw != nil {
		pi.mw.Destroy()
		pi.mw = nil
	}
}

func ReadImage2(r io.Reader) (PixurImage2, status.S) {
	mw := imagick.NewMagickWand()
	destroy := true
	defer func() {
		if destroy {
			mw.Destroy()
		}
	}()
	switch r := r.(type) {
	case *os.File:
		if err := mw.ReadImageFile(r); err != nil {
			return nil, status.InvalidArgument(err, "unable to decode image")
		}
	default:
		var b bytes.Buffer
		if _, err := io.Copy(&b, r); err != nil {
			return nil, status.InvalidArgument(err, "unable to copy image")
		}
		if err := mw.ReadImageBlob(b.Bytes()); err != nil {
			return nil, status.InvalidArgument(err, "unable to decode image")
		}
	}
	pi := &pixurImage2{mw: mw}
	destroy = false
	return pi, nil
}
