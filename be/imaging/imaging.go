package imaging

import (
	"bytes"
	"io"
	"time"

	"pixur.org/pixur/be/status"
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

const (
	// WEBM magic header
	ebmlHeader = "\x1a\x45\xdf\xa3"
)

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

var defaultimagereader func(r io.Reader) (PixurImage, status.S)
var defaultwebmreader func(r io.Reader) (PixurImage, status.S)

type rra interface {
	io.Reader
	io.ReaderAt
}

func ReadImage(r io.Reader) (PixurImage, status.S) {
	var ra rra
	switch r := r.(type) {
	case rra:
		ra = r
	default:
		var b bytes.Buffer
		if _, err := io.Copy(&b, r); err != nil {
			return nil, status.InvalidArgument(err, "unable to copy image")
		}
		ra = bytes.NewReader(b.Bytes())
	}
	firstfour := make([]byte, 4)
	if _, err := ra.ReadAt(firstfour, 0); err != nil {
		return nil, status.InvalidArgument(err, "unable to read first 4 bytes")
	}
	if bytes.Equal(firstfour, []byte(ebmlHeader)) {
		return defaultwebmreader(ra)
	}
	return defaultimagereader(ra)
}
