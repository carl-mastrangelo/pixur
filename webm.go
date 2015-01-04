package pixur

import (
	"encoding/json"
	"fmt"
	"image"
	"os"
	"os/exec"
)

const (
	maxWebmDuration = 60*2 + 1 // Two minutes, with 1 second of leeway
)

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

func fillImageConfigFromWebm(tempFile *os.File, p *Pic) (image.Image, error) {
	config, err := getWebmConfig(tempFile.Name())
	if err != nil {
		return nil, err
	}
	p.Mime = Mime_WEBM
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
