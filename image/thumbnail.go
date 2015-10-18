package image

import (
	"encoding/json"
	"fmt"
	img "image"
	"image/draw"
	"log"
	"math"
	"os"
	"os/exec"
	"time"

	"pixur.org/pixur/schema"

	"github.com/nfnt/resize"

	"image/gif"
	"image/jpeg"
	_ "image/png"
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
		}
	}

	return im, nil
}

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

func getFirstWebmFrame(filepath string) (img.Image, error) {
	cmd := exec.Command("ffmpeg",
		"-i", filepath,
		"-v", "quiet", // disable version info
		"-frames:v", "1",
		"-ss", "1", // Grab the last frame before the first second
		"-codec:v", "png",
		"-f", "image2pipe",
		"-")
	// PNG data comes across stdout
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	im, _, err := img.Decode(stdout)
	if err != nil {
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	return im, nil
}
