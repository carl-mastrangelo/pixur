package pixur

import (
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"os"
	"os/exec"

	"pixur.org/pixur/schema"

	"github.com/nfnt/resize"

	_ "image/gif"
	"image/jpeg"
	_ "image/png"
)

// TODO: maybe make this into it's own package
const (
	DefaultThumbnailWidth  = 160
	DefaultThumbnailHeight = 160

	maxWebmDuration = 60*2 + 1 // Two minutes, with 1 second of leeway
)

func FillImageConfig(f *os.File, p *schema.Pic) (image.Image, error) {
	if _, err := f.Seek(0, os.SEEK_SET); err != nil {
		return nil, err
	}

	img, imgType, err := image.Decode(f)
	if err == image.ErrFormat {
		// Try Webm
		img, err = fillImageConfigFromWebm(f, p)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		// TODO: handle this error
		p.Mime, _ = schema.FromImageFormat(imgType)
		p.Width = int64(img.Bounds().Dx())
		p.Height = int64(img.Bounds().Dy())
	}

	return img, nil
}

// TODO: interpret image rotation metadata
func MakeThumbnail(img image.Image) image.Image {
	bounds := findMaxSquare(img.Bounds())
	largeSquareImage := image.NewNRGBA(bounds)
	draw.Draw(largeSquareImage, bounds, img, bounds.Min, draw.Src)
	return resize.Resize(DefaultThumbnailWidth, DefaultThumbnailHeight, largeSquareImage,
		resize.NearestNeighbor)
}

func SaveThumbnail(img image.Image, p *schema.Pic, pixPath string) error {
	f, err := os.Create(p.ThumbnailPath(pixPath))
	if err != nil {
		return err
	}
	defer f.Close()
	return jpeg.Encode(f, img, nil)
}

func findMaxSquare(bounds image.Rectangle) image.Rectangle {
	width := bounds.Dx()
	height := bounds.Dy()
	if height < width {
		missingSpace := width - height
		return image.Rectangle{
			Min: image.Point{
				X: bounds.Min.X + missingSpace/2,
				Y: bounds.Min.Y,
			},
			Max: image.Point{
				X: bounds.Min.X + missingSpace/2 + height,
				Y: bounds.Max.Y,
			},
		}
	} else {
		missingSpace := height - width
		return image.Rectangle{
			Min: image.Point{
				X: bounds.Min.X,
				Y: bounds.Min.Y + missingSpace/2,
			},
			Max: image.Point{
				X: bounds.Max.X,
				Y: bounds.Min.Y + missingSpace/2 + width,
			},
		}
	}
}

type ffprobeConfig struct {
	Streams []ffprobeStream `json:"streams"`
	Format  ffprobeFormat   `json:"format"`
}

type ffprobeFormat struct {
	StreamCount int     `json:"nb_streams"`
	FormatName  string  `json:"format_name"`
	Duration    float64 `json:"duration,string"`
}

type ffprobeStream struct {
	CodecName string `json:"codec_name"`
	CodecType string `json:"codec_type"`
	Width     int64  `json:"width"`
	Height    int64  `json:"height"`
}

func fillImageConfigFromWebm(tempFile *os.File, p *schema.Pic) (image.Image, error) {
	config, err := getWebmConfig(tempFile.Name())
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

	return getFirstWebmFrame(tempFile.Name())
}

func getWebmConfig(filepath string) (*ffprobeConfig, error) {
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
	config := new(ffprobeConfig)
	if err := json.NewDecoder(stdout).Decode(config); err != nil {
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	if config.Format.FormatName != "matroska,webm" {
		return nil, fmt.Errorf("Only webm supported: %v", config)
	}
	if config.Format.StreamCount != 1 {
		return nil, fmt.Errorf("Only single stream webms supported: %v", config)
	}
	if config.Format.Duration <= 0 || config.Format.Duration > maxWebmDuration {
		return nil, fmt.Errorf("Invalid Duration %v", config)
	}
	// There should be only 1 stream, but use a for loop just incase.
	for _, stream := range config.Streams {
		// Add more formats as needed
		if stream.CodecName != "vp8" {
			return nil, fmt.Errorf("Only Vp8 Current supported %v", config)
		}
		/* I'm not sure what to do about other stream types like subs, but just fail until there's
		   an actual example.*/
		if stream.CodecType != "video" {
			return nil, fmt.Errorf("Only Video streams supported %v", config)
		}
	}

	return config, nil
}

func getFirstWebmFrame(filepath string) (image.Image, error) {
	cmd := exec.Command("ffmpeg",
		"-i", filepath,
		"-v", "quiet", // disable version info
		"-frames:v", "1",
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

	img, _, err := image.Decode(stdout)
	if err != nil {
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	return img, nil
}
