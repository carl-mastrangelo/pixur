package imaging

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	// this is the only outside pixur package dependency.  Avoid depending too much on schema.
	"pixur.org/pixur/be/status"
)

const (
	// WEBM magic header
	ebmlHeader = "\x1a\x45\xdf\xa3"
)

var _ PixurImage = (*ffmpegImage)(nil)

type ffmpegImage struct {
	videoFrame       image.Image
	cachedVideoFrame PixurImage
	probeResponse    *probeResponse
	duration         time.Duration
}

func (im *ffmpegImage) Format() ImageFormat {
	return DefaultWebmFormat
}

func (im *ffmpegImage) Close() {
	im.videoFrame = nil
	im.probeResponse = nil
	im.duration = 0
	if im.cachedVideoFrame != nil {
		im.cachedVideoFrame.Close()
		im.cachedVideoFrame = nil
	}
}

func (im *ffmpegImage) Dimensions() (width, height uint) {
	rectangle := im.videoFrame.Bounds()
	return uint(rectangle.Dx()), uint(rectangle.Dy())
}

func (im *ffmpegImage) Duration() (*time.Duration, status.S) {
	tim := im.duration
	return &tim, nil
}

func (im *ffmpegImage) videoFrameImage() (PixurImage, status.S) {
	if im.cachedVideoFrame == nil {
		var buf bytes.Buffer
		enc := png.Encoder{CompressionLevel: png.NoCompression}
		if err := enc.Encode(&buf, im.videoFrame); err != nil {
			return nil, status.InternalError(err, "unable to encode video frame")
		}
		im2, sts := ReadImage(bytes.NewReader(buf.Bytes()))
		if sts != nil {
			return nil, sts
		}
		im.cachedVideoFrame = im2
	}
	return im.cachedVideoFrame, nil
}

func (im *ffmpegImage) PerceptualHash0() ([]byte, []float32, status.S) {
	im2, sts := im.videoFrameImage()
	if sts != nil {
		return nil, nil, sts
	}
	return im2.PerceptualHash0()
}

func (im *ffmpegImage) Thumbnail() (PixurImage, status.S) {
	im2, sts := im.videoFrameImage()
	if sts != nil {
		return nil, sts
	}
	return im2.Thumbnail()
}

func (im *ffmpegImage) Write(io.Writer) status.S {
	return status.Unimplemented(nil, "write not supported")
}

func ffmpegDecode(r io.Reader) (*ffmpegImage, status.S) {
	var wg sync.WaitGroup

	pr, pw := io.Pipe()
	defer pr.Close()
	defer pw.Close()
	var resp *probeResponse
	var probeSts status.S
	wg.Add(1)
	go func() {
		defer wg.Done()
		resp, probeSts = ffmpegProbe(pr)
		// Throw away extra data, to not block the multiwriter
		io.Copy(ioutil.Discard, pr)
	}()

	cr, cw := io.Pipe()
	defer cr.Close()
	defer cw.Close()
	var img image.Image
	var convertSts status.S
	wg.Add(1)
	go func() {
		defer wg.Done()
		img, convertSts = ffmpegConvert(cr)
		io.Copy(ioutil.Discard, cr)
	}()

	if _, err := io.Copy(io.MultiWriter(pw, cw), r); err != nil {
		pw.CloseWithError(err)
		cw.CloseWithError(err)
		return nil, status.InvalidArgument(err, "unable to read in ffprobe/ffmpeg")
	} else {
		pw.Close()
		cw.Close()
	}
	wg.Wait()

	if probeSts != nil {
		return nil, probeSts
	}
	if convertSts != nil {
		return nil, convertSts
	}

	if sts := checkValidWebm(resp); sts != nil {
		return nil, sts
	}
	// duration was already checked in checkValidWebm
	duration, _ := parseFfmpegDuration(resp.Format.Duration)

	return &ffmpegImage{
		videoFrame:    img,
		duration:      duration,
		probeResponse: resp,
	}, nil
}

func ffmpegConvert(r io.Reader) (image.Image, status.S) {
	cmd := exec.Command(
		"ffmpeg",
		"-hide_banner",
		"-t", "1.0", // Grab the last frame before the first second
		"-i", "-", // reading from stdin
		"-frames:v", "120", // Handle up to 120fps video, then give up.
		"-codec:v", "png",
		"-compression_level", "0", // Don't bother compressing
		"-f", "image2pipe",
		"-")

	var errBuf bytes.Buffer
	cmd.Stdin = r
	cmd.Stderr = &errBuf
	// PNG data comes across stdout
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, status.InternalError(err, "unable to create pipe")
	}
	defer stdout.Close()

	if err := cmd.Start(); err != nil {
		return nil, status.InternalError(err, "unable to start ffmpeg: "+errBuf.String())
	}
	defer cmd.Process.Kill()

	im, sts := keepLastImage(stdout)
	if sts != nil {
		// This should be a deferred
		return nil, status.InternalError(sts, "unable to get sample image: "+errBuf.String())
	}
	// See explanation why in probe()
	if err := stdout.Close(); err != nil {
		return nil, status.InternalError(err, "unable to close stdout stream")
	}
	if err := cmd.Wait(); err != nil {
		return nil, status.InternalError(err, "unable to wait on ffmpeg: "+errBuf.String())
	}
	return im, nil
}

func checkValidWebm(resp *probeResponse) status.S {
	if resp.Format.FormatName != "matroska,webm" {
		return status.InvalidArgumentf(nil, "Only webm supported: %+v", *resp)
	}
	if resp.Format.StreamCount <= 0 {
		return status.InvalidArgumentf(nil, "No Streams found: %+v", *resp)
	}
	duration, sts := parseFfmpegDuration(resp.Format.Duration)
	if sts != nil {
		return sts
	}
	if duration < 0 || duration > maxWebmDuration {
		return status.InvalidArgumentf(nil, "Invalid duration: %v for %+v", duration, *resp)
	}

	var videoFound bool
	// Only check for a video stream, since we will just mute it on output.
	for _, stream := range resp.Streams {
		if stream.CodecType == "video" {
			if stream.CodecName == "vp8" || stream.CodecName == "vp9" {
				videoFound = true
				break
			} else {
				return status.InvalidArgumentf(nil, "Unsupported video type: %v", stream.CodecName)
			}
		} else if stream.CodecType == "audio" {
			// even though we don't plan on playing it, don't allow invalid types in
			if stream.CodecName != "vorbis" && stream.CodecName != "opus" {
				return status.InvalidArgumentf(nil, "Unsupported audio type: %v", stream.CodecName)
			}
		}
	}

	if !videoFound {
		return status.InvalidArgumentf(nil, "No video found: %+v", *resp)
	}

	return nil
}

func ffmpegProbe(r io.Reader) (*probeResponse, status.S) {
	cmd := exec.Command(
		"ffprobe",
		"-hide_banner",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		"-")

	// Use a buffer to avoid blocking Wait.
	var outBuf bytes.Buffer
	var errBuf bytes.Buffer

	cmd.Stdin = r
	cmd.Stderr = &errBuf
	cmd.Stdout = &outBuf

	if err := cmd.Start(); err != nil {
		return nil, status.InternalError(err, "Unable to start ffprobe: "+errBuf.String())
	}
	defer cmd.Process.Kill()

	if err := cmd.Wait(); err != nil {
		return nil, status.InternalError(err, "Unable to wait on ffprobe: "+errBuf.String())
	}

	resp := new(probeResponse)
	if err := json.NewDecoder(&outBuf).Decode(resp); err != nil {
		return nil, status.InternalError(err, "Unable to decode ffprobe json: "+errBuf.String())
	}

	return resp, nil
}

// Reads in a concatenated set of images and returns the last one.
// An error is returned if no images could be read, or the there was a
// decode error.
func keepLastFfmpegImage(r io.Reader) (image.Image, status.S) {
	maxFrames := 120
	var im image.Image
	for i := 0; i < maxFrames; i++ {
		// don't use image.Decode because it doesn't return EOF on EOF
		lastIm, err := png.Decode(r)

		if err == io.ErrUnexpectedEOF {
			break
		} else if err != nil {
			return nil, status.InternalError(err, "unable to find frame")
		}
		im = lastIm
	}

	if im == nil {
		return nil, status.InvalidArgument(nil, "unable to find frames in video file")
	}

	return im, nil
}

// parseFfmpegDuration parses the ffmpeg rational format
func parseFfmpegDuration(raw string) (time.Duration, status.S) {
	parts := strings.Split(raw, ".")
	if len(parts) != 2 {
		return 0, status.InvalidArgumentf(nil, "Bad duration %v", raw)
	}
	seconds, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, status.InvalidArgumentf(err, "Bad duration %v", raw)
	}
	if len(parts[1]) > 9 {
		parts[1] = parts[1][:9]
	} else {
		for len(parts[1]) != 9 {
			parts[1] += "0"
		}
	}
	nanos, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, status.InvalidArgumentf(err, "Bad duration %v", raw)
	}

	dur := time.Duration(seconds)*time.Second + time.Duration(nanos)*time.Nanosecond
	return dur, nil
}

type probeResponse struct {
	Streams []probeStream `json:"streams"`
	Format  probeFormat   `json:"format"`
}

type probeFormat struct {
	StreamCount int               `json:"nb_streams"`
	FormatName  string            `json:"format_name"`
	Duration    string            `json:"duration"`
	Tags        map[string]string `json:"tags"`
}

type probeStream struct {
	CodecName string `json:"codec_name"`
	CodecType string `json:"codec_type"`
	Width     int64  `json:"width"`
	Height    int64  `json:"height"`
}
