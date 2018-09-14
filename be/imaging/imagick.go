package imaging

import (
	"bytes"
	"encoding/binary"
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
	DefaultGifFormat  ImageFormat = "GIF"
	DefaultJpegFormat ImageFormat = "JPEG"
	DefaultPngFormat  ImageFormat = "PNG"
	DefaultWebmFormat ImageFormat = "WEBM"
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

// IsJpeg returns true if the type of this image is a WEBM.
func (f ImageFormat) IsWebm() bool {
	return f == "WEBM"
}

type PixurImage interface {
	Format() ImageFormat
	Dimensions() (width, height uint)
	// Duration returns how long the image is animated for, or nil if the image is not animated.
	// It may return 0s.  In the future, this could also include a histogram
	Duration() (*time.Duration, status.S)

	Thumbnail() (PixurImage, status.S)

	PerceptualHash0() ([]byte, []float32, status.S)

	Write(io.Writer) status.S

	Close()
}

var _ PixurImage = (*imagickImage)(nil)

type imagickImage struct {
	mw *imagick.MagickWand
}

func (pi *imagickImage) PerceptualHash0() ([]byte, []float32, status.S) {
	newmw := pi.mw.Clone()
	defer newmw.Destroy()
	newmw.ResetIterator()

	// Intuitively it seems this transform is needed.  (it's needed for the normal thumbnail tansform
	// however, some random tests I did show it makes the results worse.  Also this matches the prev
	// algorithm
	//newmw.TransformImageColorspace(imagick.COLORSPACE_RGB)
	newmw.TransformImageColorspace(imagick.COLORSPACE_SRGB)
	if err := newmw.ResizeImage(dctSize, dctSize, imagick.FILTER_LANCZOS2_SHARP, 1); err != nil {
		return nil, nil, status.InternalError(err, "can't resize")
	}

	// TODO: maybe do this in LAB?  Just using GRAY 'cuz that's how it was before.
	newmw.TransformImageColorspace(imagick.COLORSPACE_GRAY)

	it := newmw.NewPixelIterator()
	defer it.Destroy()
	var grays [][]float64 = make([][]float64, int(newmw.GetImageHeight()))
	for y := 0; y < len(grays); y++ {
		row := it.GetNextIteratorRow()
		if len(row) != dctSize {
			panic(len(row))
		}
		for _, pix := range row {
			g := (255 * pix.GetGreen()) - 128 // [-128.0 - 127.0]
			grays[y] = append(grays[y], g)
		}
	}
	dcts := dct2d(grays)
	hash, inputs := phash0(dcts)
	outputs := make([]float32, len(inputs))
	for i, input := range inputs {
		outputs[i] = float32(input)
	}
	hashBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(hashBytes, hash)

	return hashBytes, outputs, nil
}

func (pi *imagickImage) Write(w io.Writer) status.S {
	defer pi.mw.ResetIterator()
	// TODO: maybe make this work with GIF?  I don't think there is a case that Pixur wants to write
	// back out a non-thumbnail.  All thumbnails are single image.
	switch w := w.(type) {
	case *os.File:
		if err := pi.mw.WriteImageFile(w); err != nil {
			return status.InternalError(err, "can't write image file")
		}
	default:
		if _, err := w.Write(pi.mw.GetImageBlob()); err != nil {
			return status.InternalError(err, "can't write image")
		}
	}

	return nil
}

func (pi *imagickImage) Thumbnail() (PixurImage, status.S) {
	defer pi.mw.ResetIterator()
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
	destroy := true
	defer func() {
		if destroy {
			newmw.Destroy()
		}
	}()
	defer newmw.ResetIterator()

	// Some PNGs have an `oFFs` section that messes with the crop
	if err := newmw.SetImagePage(w, h, 0, 0); err != nil {
		return nil, status.InternalError(err, "unable to repage image")
	}

	if err := newmw.CropImage(neww, newh, x, y); err != nil {
		return nil, status.InternalError(err, "unable to crop thumbnail")
	}

	side := uint(thumbnailSquareSize)
	newmw.TransformImageColorspace(imagick.COLORSPACE_RGB)
	if err := newmw.ResizeImage(side, side, imagick.FILTER_CATROM, 1); err != nil {
		return nil, status.InternalError(err, "unable to resize thumbnail")
	}
	newmw.TransformImageColorspace(imagick.COLORSPACE_SRGB)

	// Reset the image page geometry back after thumbnailing, or else it can get preserved. This
	// converts an image identified as
	// img.gif GIF 160x160 160x663+0+251 8-bit sRGB 256c 6539B 0.000u 0:00.000
	// to
	// img.gif GIF 160x160 160x160+0+0 8-bit sRGB 256c 6539B 0.000u 0:00.000
	newmw.SetImagePage(side, side, 0, 0)

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

	switch format := pi.Format(); {
	case format.IsJpeg():
		newmw.SetImageCompressionQuality(90)
	default:
		pw := imagick.NewPixelWand()
		defer pw.Destroy()
		if !pw.SetColor("white") {
			return nil, status.InternalError(nil, "unable to find background color white")
		}
		if err := newmw.SetImageBackgroundColor(pw); err != nil {
			return nil, status.InternalError(err, "unable to set background color")
		}
		if err := newmw.SetImageAlphaChannel(imagick.ALPHA_CHANNEL_REMOVE); err != nil {
			return nil, status.InternalError(err, "unable to remove alpha channel")
		}
		if err := newmw.SetImageFormat(string(DefaultJpegFormat)); err != nil {
			return nil, status.InternalError(err, "unable to set format")
		}
		// png, gif, and (webm via png) always get full quality.   This is not so big for thumbnails
		// and the decode speed of jpeg is fast.
		newmw.SetImageCompressionQuality(100)
	}
	if err := newmw.SetOption("jpeg:sampling-factor", "1x1,1x1,1x1"); err != nil {
		return nil, status.InternalError(err, "unable to set sampling factor")
	}

	// TODO:trim profiles (keep colorspace?), apply orientation,
	newpi := &imagickImage{
		mw: newmw,
	}
	destroy = false
	return newpi, nil
}

func (pi *imagickImage) Format() ImageFormat {
	return ImageFormat(pi.mw.GetImageFormat())
}

func (pi *imagickImage) Dimensions() (w uint, h uint) {
	return pi.mw.GetImageWidth(), pi.mw.GetImageHeight()
}

func (pi *imagickImage) Duration() (*time.Duration, status.S) {
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
	defer pi.mw.ResetIterator()
	const tickDuration = time.Second / gifTicksPerSecond
	const delayTicksMax = math.MaxInt64 / int64(tickDuration)
	for pi.mw.NextImage() {
		delayTicks := int64(pi.mw.GetImageDelay())
		if delayTicks < 0 || delayTicks > delayTicksMax {
			return nil, status.InvalidArgument(nil, "delayTicks would overflow", delayTicks)
		}
		// Browsers treat low tick count by rounding up tick count as described in
		// http://nullsleep.tumblr.com/post/16524517190/animated-gif-minimum-frame-delay-browser
		// Prefer the Firefox / Chrome interpretation.
		var roundedDelayTicks int64
		switch delayTicks {
		case 0:
			roundedDelayTicks = 10
		case 1:
			roundedDelayTicks = 10
		default:
			roundedDelayTicks = delayTicks
		}
		// this should always be positive
		delayTickDuration := time.Duration(roundedDelayTicks) * tickDuration
		if d += delayTickDuration; d < 0 {
			return nil, status.InvalidArgument(nil, "duration overflow", d)
		}
	}

	return &d, nil
}

func (pi *imagickImage) Close() {
	if pi.mw != nil {
		pi.mw.Destroy()
		pi.mw = nil
	}
}

func ReadImage(r io.Reader) (PixurImage, status.S) {
	mw := imagick.NewMagickWand()
	destroy := true
	defer func() {
		if destroy {
			mw.Destroy()
		}
	}()
	defer mw.ResetIterator()
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
	pi := &imagickImage{mw: mw}
	destroy = false
	return pi, nil
}
