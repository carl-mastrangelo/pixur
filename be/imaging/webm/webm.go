package webm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type WebmErr struct {
	Err       error
	DebugInfo string
}

type WebmImage struct {
	SubImager

	Duration time.Duration

	// Metadata, not comprehensive
	Tags map[string]string
}

func (e *WebmErr) Error() string {
	return e.Err.Error()
}

// copy of imaging.SubImager, to avoid a dependency loop
type SubImager interface {
	image.Image
	SubImage(image.Rectangle) image.Image
}

const (
	ebmlHeader = "\x1a\x45\xdf\xa3"

	// Two minutes, with 1 second of leeway
	maxWebmDuration = time.Duration(60*2+1) * time.Second
)

func decodeConfig(r io.Reader) (image.Config, error) {
	// TODO: implement
	return image.Config{}, nil
}

// Go's standard library makes this difficult to do cleanly.  image.Decode
// internally wraps any reader that doesn't implement Peek() with an
// unexported type.  This means decode() cannot just cast to a more appropriate
// type.  This needs to read over the input twice, once for ffprobe and also
// ffmpeg.  Ideally, we would probe first and decide if we want to do the full
// conversion.  But, since we only get one shot, we have to copy the input.
// This means either:
//   A. buffering the whole input in memory
//   B. buffering the input to disk
//   C. feeding both ffprobe and ffmpeg in lock step
// A is not feasible.  B is possible, but will very likely result in copying
// between partitions.  C is crummy too, since synchronization is now involved.
func decode(r io.Reader) (image.Image, error) {
	var wg sync.WaitGroup

	pr, pw := io.Pipe()
	defer pr.Close()
	defer pw.Close()
	var resp *probeResponse
	var probeErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		resp, probeErr = probe(pr)
		// Throw away extra data, to not block the multiwriter
		io.Copy(ioutil.Discard, pr)
	}()

	cr, cw := io.Pipe()
	defer cr.Close()
	defer cw.Close()
	var img SubImager
	var convertErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		img, convertErr = convert(cr)
		io.Copy(ioutil.Discard, cr)
	}()

	if _, err := io.Copy(io.MultiWriter(pw, cw), r); err != nil {
		pw.CloseWithError(err)
		cw.CloseWithError(err)
		return nil, err
	} else {
		pw.Close()
		cw.Close()
	}
	wg.Wait()

	if probeErr != nil {
		return nil, probeErr
	}
	if convertErr != nil {
		return nil, convertErr
	}

	if err := checkValidWebm(*resp); err != nil {
		return nil, err
	}
	// duration was already checked in checkValidWebm
	duration, _ := parseDuration(resp.Format.Duration)

	return &WebmImage{
		SubImager: img,
		Duration:  duration,
		Tags:      resp.Format.Tags,
	}, nil
}

func convert(r io.Reader) (SubImager, error) {
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
		return nil, err
	}
	defer stdout.Close()

	if err := cmd.Start(); err != nil {
		return nil, &WebmErr{Err: err, DebugInfo: errBuf.String()}
	}
	defer cmd.Process.Kill()

	im, err := keepLastImage(stdout)
	if err != nil {
		return nil, &WebmErr{Err: err, DebugInfo: errBuf.String()}
	}
	// See explanation why in probe()
	if err := stdout.Close(); err != nil {
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		return nil, &WebmErr{Err: err, DebugInfo: errBuf.String()}
	}
	return im.(SubImager), nil
}

// Reads in a concatenated set of images and returns the last one.
// An error is returned if no images could be read, or the there was a
// decode error.
func keepLastImage(r io.Reader) (SubImager, error) {
	maxFrames := 120
	var im SubImager
	for i := 0; i < maxFrames; i++ {
		// don't use image.Decode because it doesn't return EOF on EOF
		lastIm, err := png.Decode(r)

		if err == io.ErrUnexpectedEOF {
			break
		} else if err != nil {
			return nil, err
		}
		im = lastIm.(SubImager)
	}

	if im == nil {
		return nil, fmt.Errorf("No frames in webm")
	}

	return im, nil
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

func probe(r io.Reader) (*probeResponse, error) {
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
		return nil, &WebmErr{Err: err, DebugInfo: errBuf.String()}
	}
	defer cmd.Process.Kill()

	if err := cmd.Wait(); err != nil {
		return nil, &WebmErr{Err: err, DebugInfo: errBuf.String()}
	}

	resp := new(probeResponse)
	if err := json.NewDecoder(&outBuf).Decode(resp); err != nil {
		return nil, &WebmErr{Err: err, DebugInfo: errBuf.String()}
	}

	return resp, nil
}

func checkValidWebm(resp probeResponse) error {
	if resp.Format.FormatName != "matroska,webm" {
		return fmt.Errorf("Only webm supported: %+v", resp)
	}
	if resp.Format.StreamCount <= 0 {
		return fmt.Errorf("No Streams found: %+v", resp)
	}
	duration, err := parseDuration(resp.Format.Duration)
	if err != nil {
		return err
	}
	if duration < 0 || duration > maxWebmDuration {
		return fmt.Errorf("Invalid duration %+v", resp)
	}

	var videoFound bool
	// Only check for a video stream, since we will just mute it on output.
	for _, stream := range resp.Streams {
		if stream.CodecType == "video" {
			if stream.CodecName == "vp8" || stream.CodecName == "vp9" {
				videoFound = true
				break
			} else {
				return fmt.Errorf("Unsupported video type %v", stream.CodecName)
			}
		} else if stream.CodecType == "audio" {
			// even though we don't plan on playing it, don't allow invalid types in
			if stream.CodecName != "vorbis" && stream.CodecName != "opus" {
				return fmt.Errorf("Unsupported audio type %v", stream.CodecName)
			}
		}
	}

	if !videoFound {
		return fmt.Errorf("No video found %+v", resp)
	}

	return nil
}

// parseDuration parses the ffmpeg rational format
func parseDuration(raw string) (time.Duration, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 2 {
		return 0, fmt.Errorf("Bad duration %v", raw)
	}
	seconds, err1 := strconv.ParseInt(parts[0], 10, 64)
	micros, err2 := strconv.ParseInt(parts[1], 10, 64)
	if err1 != nil || err2 != nil {
		return 0, fmt.Errorf("Bad duration %v", raw)
	}

	dur := time.Duration(seconds)*time.Second + time.Duration(micros*1000)*time.Nanosecond
	return dur, nil
}

func init() {
	image.RegisterFormat("webm", ebmlHeader, decode, decodeConfig)
}
