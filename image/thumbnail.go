package image

import (
	"encoding/json"
	"fmt"
	img "image"
	"image/draw"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"time"

	"pixur.org/pixur/schema"

	"github.com/nfnt/resize"

	"image/gif"
	"image/jpeg"
	"image/png"
)

// TODO: maybe make this into it's own package
const (
	DefaultThumbnailWidth  = 160
	DefaultThumbnailHeight = 160

	maxWebmDuration = 60*2 + 1 // Two minutes, with 1 second of leeway
)

type BadWebmFormatErr struct {
	error
}

func FillImageConfig(f *os.File, p *schema.Pic) (img.Image, error) {
	if _, err := f.Seek(0, os.SEEK_SET); err != nil {
		return nil, err
	}
	defer f.Seek(0, os.SEEK_SET)

	im, imgType, err := img.Decode(f)
	if err == img.ErrFormat {
		// Try Webm
		im, err = fillImageConfigFromWebm(f, p)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		// TODO: handle this error
		p.Mime, _ = schema.FromImageFormat(imgType)
		p.Width = int64(im.Bounds().Dx())
		p.Height = int64(im.Bounds().Dy())
	}

	if p.Mime == schema.Pic_GIF {
		if _, err := f.Seek(0, os.SEEK_SET); err != nil {
			return nil, err
		}

		GIF, err := gif.DecodeAll(f)
		if err != nil {
			return nil, err
		}
		// Ignore gifs that have only one frame
		if len(GIF.Delay) > 1 {
			p.AnimationInfo = &schema.AnimationInfo{
				Duration: GetGifDuration(GIF),
			}
			// TODO: maybe skip the first second of frames like webm
		}
	}

	return im, nil
}

// Returns the duration of this gif.  It will be different per browser,
// with inaccuracies usually in the 0/100 and 1/100 delays (which are
// rounded up to 10/100).  http://nullsleep.tumblr.com/post/16524517190/
// animated-gif-minimum-frame-delay-browser describes the delays for
// common browsers.  An idea to solve this is to keep a histogram of the
// short delay frames (from 0-5 hundredths) and allow the browser js to
// reinterpret the duration.
// TODO: add tests for this
func GetGifDuration(g *gif.GIF) *schema.Duration {
	var duration time.Duration
	// TODO: check for overflow
	// each delay unit is 1/100 of a second
	for _, frameHundredths := range g.Delay {
		duration += time.Millisecond * time.Duration(10*frameHundredths)
	}

	return schema.FromDuration(duration)
}

// TODO: interpret image rotation metadata
func MakeThumbnail(im img.Image) img.Image {
	bounds := findMaxSquare(im.Bounds())
	largeSquareImage := img.NewNRGBA(bounds)
	draw.Draw(largeSquareImage, bounds, im, bounds.Min, draw.Src)
	return resize.Resize(DefaultThumbnailWidth, DefaultThumbnailHeight, largeSquareImage,
		resize.NearestNeighbor)
}

func SaveThumbnail(im img.Image, p *schema.Pic, pixPath string) error {
	f, err := os.Create(p.ThumbnailPath(pixPath))
	if err != nil {
		return err
	}
	defer f.Close()
	return jpeg.Encode(f, im, nil)
}

func findMaxSquare(bounds img.Rectangle) img.Rectangle {
	width := bounds.Dx()
	height := bounds.Dy()
	if height < width {
		missingSpace := width - height
		return img.Rectangle{
			Min: img.Point{
				X: bounds.Min.X + missingSpace/2,
				Y: bounds.Min.Y,
			},
			Max: img.Point{
				X: bounds.Min.X + missingSpace/2 + height,
				Y: bounds.Max.Y,
			},
		}
	} else {
		missingSpace := height - width
		return img.Rectangle{
			Min: img.Point{
				X: bounds.Min.X,
				Y: bounds.Min.Y + missingSpace/2,
			},
			Max: img.Point{
				X: bounds.Max.X,
				Y: bounds.Min.Y + missingSpace/2 + width,
			},
		}
	}
}

type FFprobeConfig struct {
	Streams []FFprobeStream `json:"streams"`
	Format  FFprobeFormat   `json:"format"`
}

type FFprobeFormat struct {
	StreamCount int     `json:"nb_streams"`
	FormatName  string  `json:"format_name"`
	Duration    float64 `json:"duration,string"`
}

type FFprobeStream struct {
	CodecName string `json:"codec_name"`
	CodecType string `json:"codec_type"`
	Width     int64  `json:"width"`
	Height    int64  `json:"height"`
}

func fillImageConfigFromWebm(tempFile *os.File, p *schema.Pic) (img.Image, error) {
	config, err := GetWebmConfig(tempFile.Name())
	if err != nil {
		return nil, err
	}
	p.Mime = schema.Pic_WEBM
	// Handle the 0 and 1 case
	for _, stream := range config.Streams {
		p.Width = stream.Width
		p.Height = stream.Height
		break
	}

	if dur, success := ConvertFloatDuration(config.Format.Duration); success {
		if dur >= time.Nanosecond {
			p.AnimationInfo = &schema.AnimationInfo{
				Duration: schema.FromDuration(dur),
			}
		}
	} else {
		log.Println("Invalid duration from ffmpeg: ", config.Format.Duration)
	}

	return getFirstWebmFrame(tempFile.Name())
}

func ConvertFloatDuration(seconds float64) (time.Duration, bool) {
	if math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds > math.MaxInt64 {
		return time.Duration(0), false
	}
	if seconds < 0 {
		return time.Duration(0), false
	}
	return time.Duration(seconds * 1e9), true
}

func GetWebmConfig(filepath string) (*FFprobeConfig, error) {
	cmd := exec.Command("ffprobe",
		"-print_format", "json",
		"-v", "quiet", // disable version info
		"-show_format",
		"-show_streams",
		filepath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	config := new(FFprobeConfig)
	if err := json.NewDecoder(stdout).Decode(config); err != nil {
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	if config.Format.FormatName != "matroska,webm" {
		return nil, fmt.Errorf("Only webm supported: %v", config)
	}
	if config.Format.StreamCount == 0 {
		return nil, fmt.Errorf("No Streams found: %v", config)
	}
	if config.Format.Duration < 0 || config.Format.Duration > maxWebmDuration {
		return nil, fmt.Errorf("Invalid Duration %v", config)
	}

	var videoFound bool
	// Only check for a video stream, since we will just mute it on output.
	for _, stream := range config.Streams {
		if stream.CodecType == "video" {
			if stream.CodecName == "vp8" || stream.CodecName == "vp9" {
				videoFound = true
				break
			}
		}
	}

	if !videoFound {
		return nil, &BadWebmFormatErr{fmt.Errorf("Bad Video %v", config)}
	}

	return config, nil
}

// The Idea here is to read the first 1 second of video data, and stop after
// that.  The last frame received will be the thumbnail.  Normally the first
// frame of video will be an intro or darker frame.  This is undesirable, so
// we give the video 1 second of warm up time.
// This doesn't use the -ss argument, because if the video is less than one
// second, no output will be produced.  In general, if there is not a frame
// after the first duration, no output will be produced.  If there was a video
// with a single frame that had a duration of 2 seconds, no output would be
// produced.  This also means that we can't just seek to some percentage of
// the way through the video and get a frame.
// This takes a different approach: output every frame, stopping after the
// first second of video time.  Then, keep the last frame produced.  This
// takes advantage of PNGs being self contained, allowing this to read the
// concatenated image output produced by ffmpeg.  Thus, videos shorter than
// 1 second will produce *something*, and videos longer than 1 don't need to
// be completely parsed.
// TODO: This is pretty slow, find a faster way.
func getFirstWebmFrame(filepath string) (img.Image, error) {
	cmd := exec.Command("ffmpeg",
		"-i", filepath,
		"-v", "quiet", // disable version info
		"-t", "1.0", // Grab the last frame before the first second
		"-frames:v", "120", // Handle up to 120fps video, then give up.
		"-codec:v", "png",
		"-compression_level", "0", // Don't bother compressing
		"-f", "image2pipe",
		"-")
	// PNG data comes across stdout
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	defer stdout.Close()
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	defer cmd.Process.Kill()
	// Avoid checking more than 120 frames
	maxFrames := 120
	var im img.Image

	for i := 0; i < maxFrames; i++ {
		// Can't use image.Decode because it reads too far ahead.
		lastIm, err := png.Decode(stdout)
		if err == io.ErrUnexpectedEOF {
			break
		} else if err != nil {
			return nil, err
		}
		im = lastIm
	}
	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	if im == nil {
		return nil, &BadWebmFormatErr{fmt.Errorf("No frames in webm")}
	}

	return im, nil
}
