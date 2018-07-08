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
	DefaultGifFormat  ImageFormat = "GIF"
	DefaultJpegFormat ImageFormat = "JPEG"
	DefaultPngFormat  ImageFormat = "PNG"
)

const gifTicksPerSecond = 100

func init() {
	imagick.Initialize()
	// never call Terminate
}

// a list of all formats: mw.QueryFormats("*")

// ImageFormat is the string format of the image
type ImageFormat string

func (f ImageFormat) IsGif() bool {
	return f == "GIF" || f == "GIF87"
}

func (f ImageFormat) IsJpeg() bool {
	return f == "JPE" || f == "JPEG" || f == "JPG"
}

func (f ImageFormat) IsPng() bool {
	// These are the types imagick says it supports: PNG PNG00 PNG24 PNG32 PNG48 PNG64 PNG8.  I am
	// only including the ones I know.
	return f == "PNG" || f == "PNG24" || f == "PNG32" || f == "PNG8"
}

type PixurImage2 interface {
	Format() ImageFormat
	Dimensions() (width, height int64, sts status.S)
	// In the future, this could also include a histogram
	Duration() (*time.Duration, status.S)
	
	Thumbnail(width, height int64, format ImageFormat) (PixurImage2, status.S)
	
	Write(io.Writer) status.S

	Close()
}

var _ PixurImage2 = (*pixurImage2)(nil)

type pixurImage2 struct {
	mw *imagick.MagickWand
}

func (pi *pixurImage2) Write(io.Writer) status.S {
  return nil
}

func (pi *pixurImage2) Thumbnail(width, height int64, format ImageFormat) (PixurImage2, status.S) {
  newmw := pi.mw.Clone()
  // TODO: crop, set colorspace, trim profiles, optimize
  newpi := &pixurImage2{
    mw: newmw,
  }
  return newpi, nil
}

func (pi *pixurImage2) Format() ImageFormat {
	return ImageFormat(pi.mw.GetImageFormat())
}

func (pi *pixurImage2) Dimensions() (int64, int64, status.S) {
	w, h, _, _, err := pi.mw.GetImagePage()
	if err != nil {
		return 0, 0, status.InternalError(err, "unable to get image dimensions")
	}
	return int64(w), int64(h), nil
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
