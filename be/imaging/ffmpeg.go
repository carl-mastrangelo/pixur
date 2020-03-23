package imaging

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	// this is the only outside pixur package dependency.  Avoid depending too much on schema.
	"pixur.org/pixur/be/status"
)

func init() {
	defaultvideoreader = func(ctx context.Context, r io.Reader) (PixurImage, status.S) {
		return ffmpegDecode(ctx, r)
	}
}

var _ PixurImage = (*ffmpegImage)(nil)

type ffmpegImage struct {
	ctx              context.Context
	format           ImageFormat
	videoFrame       image.Image
	cachedVideoFrame PixurImage
	probeResponse    *probeResponse
	duration         time.Duration
}

func (im *ffmpegImage) Format() ImageFormat {
	return im.format
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
	for _, s := range im.probeResponse.Streams {
		if s.Width > 0 || s.Height > 0 {
			return uint(s.Width), uint(s.Height)
		}
	}
	// this should be impossible since it previously validated.
	panic("missing video")
}

func (im *ffmpegImage) Duration() (*time.Duration, status.S) {
	tim := im.duration
	return &tim, nil
}

func (im *ffmpegImage) videoFrameImage() (PixurImage, status.S) {
	if im.videoFrame == nil {
		return nil, status.InvalidArgument(nil, "can't get image frame")
	}
	if im.cachedVideoFrame == nil {
		var buf bytes.Buffer
		enc := png.Encoder{CompressionLevel: png.NoCompression}
		if err := enc.Encode(&buf, im.videoFrame); err != nil {
			return nil, status.Internal(err, "unable to encode video frame")
		}
		im2, sts := ReadImage(im.ctx, bytes.NewReader(buf.Bytes()))
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

// ConvertVideo converts from a source video to a destination.  dstName has to be a string
// because ffmpeg wants to seek the output MP4 file to move the atoms around (like qt-faststart
// does).
func ConvertVideo(
	ctx context.Context, dstFmt ImageFormat, dst *os.File, r io.Reader) (
	_ PixurImage, stscap status.S) {
	if pos, err := dst.Seek(0, os.SEEK_CUR); err != nil {
		return nil, status.Internal(err, "can't seek file")
	} else if pos != 0 {
		// We are going to overwrite it, so being anywhere but the start would be confusing.
		return nil, status.InvalidArgument(err, "file pos must be at beginning")
	}

	args := []string{"-hide_banner", "-i", "-"}
	switch {
	case dstFmt.IsWebm():
		args = append(args, "-codec:v", "libvpx", "-crf", "22", "-b:v", "1M")
		// Normalize to even numbers because libvpx doesn't support odd dims.
		args = append(args, "-vf", "pad=width=ceil(iw/2)*2:height=ceil(ih/2)*2")
		args = append(args, "-codec:a", "libvorbis")
		args = append(args, "-f", "webm")
	case dstFmt.IsMp4():
		args = append(args, "-codec:v", "libx264", "-crf", "24", "-b:v", "1M")
		// The aac encoder is bad, but my ipad only seems to work with.   Maybe use libfaac2 later.
		args = append(args, "-codec:a", "aac")
		args = append(args, "-f", "mp4")
	default:
		return nil, status.InvalidArgument(nil, "unsupported file", dstFmt)
	}
	args = append(args, "-y", dst.Name())
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)

	var errBuf bytes.Buffer
	cmd.Stdin = r
	cmd.Stderr = &errBuf

	if err := cmd.Start(); err != nil {
		return nil, status.Internal(err, "unable to start ffmpeg: "+errBuf.String())
	}
	kill := true
	defer func() {
		if !kill {
			return
		}
		if err := cmd.Process.Kill(); err != nil {
			status.ReplaceOrSuppress(&stscap, status.Internal(err, "can't kill ffmpeg"))
		}
	}()

	if err := cmd.Wait(); err != nil {
		return nil, status.Internal(err, "unable to wait on ffmpeg: "+errBuf.String())
	}
	kill = false

	resp, probeSts := ffmpegProbe(ctx, dst)
	if probeSts != nil {
		return nil, probeSts
	}
	if _, err := dst.Seek(0, os.SEEK_SET); err != nil {
		return nil, status.Internal(err, "can't seek file")
	}

	format, sts := checkValidVideo(resp)
	if sts != nil {
		return nil, sts
	}
	// duration was already checked in checkValidVideo
	duration, _ := parseFfmpegDuration(resp.Format.Duration)

	return &ffmpegImage{
		ctx:           ctx,
		format:        ImageFormat(format),
		videoFrame:    nil,
		duration:      duration,
		probeResponse: resp,
	}, nil
}

func (im *ffmpegImage) Write(io.Writer) status.S {
	return status.Unimplemented(nil, "write not supported")
}

func ffmpegDecode(ctx context.Context, r io.Reader) (_ *ffmpegImage, stscap status.S) {
	var wg sync.WaitGroup

	pr, pw := io.Pipe()
	defer func() {
		if err := pr.Close(); err != nil {
			status.ReplaceOrSuppress(&stscap, status.Internal(err, "can't close pipe reader"))
		}
		if err := pw.Close(); err != nil {
			status.ReplaceOrSuppress(&stscap, status.Internal(err, "can't close pipe writer"))
		}
	}()
	var resp *probeResponse
	var probeSts status.S
	wg.Add(1)
	go func() {
		defer wg.Done()
		resp, probeSts = ffmpegProbe(ctx, pr)
		// Throw away extra data, to not block the multiwriter
		io.Copy(ioutil.Discard, pr)
	}()

	cr, cw := io.Pipe()
	defer func() {
		if err := cr.Close(); err != nil {
			status.ReplaceOrSuppress(&stscap, status.Internal(err, "can't close pipe reader"))
		}
		if err := cw.Close(); err != nil {
			status.ReplaceOrSuppress(&stscap, status.Internal(err, "can't close pipe writer"))
		}
	}()
	var img image.Image
	var convertSts status.S
	wg.Add(1)
	go func() {
		defer wg.Done()
		img, convertSts = ffmpegConvert(ctx, cr)
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

	format, sts := checkValidVideo(resp)
	if sts != nil {
		return nil, sts
	}
	// duration was already checked in checkValidVideo
	duration, _ := parseFfmpegDuration(resp.Format.Duration)

	return &ffmpegImage{
		ctx:           ctx,
		format:        ImageFormat(format),
		videoFrame:    img,
		duration:      duration,
		probeResponse: resp,
	}, nil
}

func ffmpegConvert(ctx context.Context, r io.Reader) (_ image.Image, stscap status.S) {
	cmd := exec.CommandContext(
		ctx,
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
		return nil, status.Internal(err, "unable to create pipe")
	}
	close_ := true
	defer func() {
		if !close_ {
			return
		}
		if err := stdout.Close(); err != nil {
			status.ReplaceOrSuppress(&stscap, status.Internal(err, "can't close stdout"))
		}
	}()

	if err := cmd.Start(); err != nil {
		return nil, status.Internal(err, "unable to start ffmpeg: "+errBuf.String())
	}
	kill := true
	defer func() {
		if !kill {
			return
		}
		if err := cmd.Process.Kill(); err != nil {
			status.ReplaceOrSuppress(&stscap, status.Internal(err, "can't kill ffmpeg"))
		}
	}()

	im, sts := keepLastImage(stdout)
	if sts != nil {
		// This should be a deferred
		return nil, status.Internal(sts, "unable to get sample image: "+errBuf.String())
	}
	// discard any remaining frames.  This should not happen, but I don't want to risk cmd.Wait()
	// hanging.
	if _, err := io.Copy(ioutil.Discard, stdout); err != nil {
		return nil, status.Internal(err, "unable to discard excess frames: "+errBuf.String())
	}

	close_ = false // Wait causes the close to happen
	if err := cmd.Wait(); err != nil {
		return nil, status.Internal(err, "unable to wait on ffmpeg: "+errBuf.String())
	}
	kill = false
	return im, nil
}

func checkValidVideo(resp *probeResponse) (string, status.S) {
	var supportedVideo map[string]bool
	var supportedAudio map[string]bool
	var format string
	switch resp.Format.FormatName {
	case "matroska,webm":
		supportedVideo = map[string]bool{"vp8": true, "vp9": true}
		supportedAudio = map[string]bool{"vorbis": true, "opus": true}
		format = string(DefaultWebmFormat)
	case "mov,mp4,m4a,3gp,3g2,mj2":
		supportedVideo = map[string]bool{"h264": true}
		supportedAudio = map[string]bool{"aac": true}
		format = string(DefaultMp4Format)
	default:
		return "", status.InvalidArgumentf(nil, "Only webm/mp4 supported: %+v", *resp)
	}
	if resp.Format.StreamCount <= 0 {
		return "", status.InvalidArgumentf(nil, "No Streams found: %+v", *resp)
	}
	duration, sts := parseFfmpegDuration(resp.Format.Duration)
	if sts != nil {
		return "", sts
	}
	if duration < 0 {
		return "", status.InvalidArgumentf(nil, "Invalid duration: %v for %+v", duration, *resp)
	}

	var videoFound bool
	// Only check for a video stream, since we will just mute it on output.
	for _, stream := range resp.Streams {
		if stream.CodecType == "video" {
			if supportedVideo[stream.CodecName] {
				videoFound = true
				break
			} else {
				return "", status.InvalidArgumentf(nil, "Unsupported video type: %v", stream.CodecName)
			}
		} else if stream.CodecType == "audio" {
			// even though we don't plan on playing it, don't allow invalid types in
			if !supportedAudio[stream.CodecName] {
				return "", status.InvalidArgumentf(nil, "Unsupported audio type: %v", stream.CodecName)
			}
		}
	}

	if !videoFound {
		return "", status.InvalidArgumentf(nil, "No video found: %+v", *resp)
	}

	return format, nil
}

func ffmpegProbe(ctx context.Context, r io.Reader) (_ *probeResponse, stscap status.S) {
	cmd := exec.CommandContext(
		ctx,
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
		return nil, status.Internal(err, "Unable to start ffprobe: "+errBuf.String())
	}
	kill := true
	defer func() {
		if !kill {
			return
		}
		if err := cmd.Process.Kill(); err != nil {
			status.ReplaceOrSuppress(&stscap, status.Internal(err, "can't kill ffprobe"))
		}
	}()

	if err := cmd.Wait(); err != nil {
		return nil, status.Internal(err, "Unable to wait on ffprobe: "+errBuf.String())
	}
	kill = false

	resp := new(probeResponse)
	if err := json.NewDecoder(&outBuf).Decode(resp); err != nil {
		return nil, status.Internal(err, "Unable to decode ffprobe json: "+errBuf.String())
	}

	return resp, nil
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

// Reads in a concatenated set of images and returns the last one.
// An error is returned if no images could be read, or the there was a
// decode error.
func keepLastImage(r io.Reader) (image.Image, status.S) {
	maxFrames := 120
	var im image.Image
	for i := 0; i < maxFrames; i++ {
		// don't use image.Decode because it doesn't return EOF on EOF
		lastIm, err := png.Decode(r)

		if err == io.ErrUnexpectedEOF {
			if im == nil {
				return nil, status.InvalidArgument(err, "unable to find frames in video file")
			} else {
				return im, nil
			}
		} else if err != nil {
			return nil, status.Internal(err, "unable to decode png image")
		}
		im = lastIm
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
