package imaging // import "pixur.org/pixur/be/imaging"

import (
	"encoding/json"
	"image"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"time"

	"github.com/golang/protobuf/ptypes"
	durpb "github.com/golang/protobuf/ptypes/duration"
	"github.com/nfnt/resize"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/status"
)

// TODO: maybe make this into it's own package
const (
	DefaultThumbnailWidth  = 160
	DefaultThumbnailHeight = 160

	maxWebmDuration = time.Second * time.Duration(60*2+1) // Two minutes, with 1 second of leeway
)

// TODO: delete this type
type BadWebmFormatErr struct {
	error
}

type PixurImageOld struct {
	SubImager
	Mime          schema.Pic_Mime
	AnimationInfo *schema.AnimationInfo
	// Metadata, not comprehensive
	Tags map[string]string
}

func ReadImageOld(r *io.SectionReader) (*PixurImageOld, status.S) {
	im, name, err := image.Decode(r)
	if err != nil {
		return nil, status.InvalidArgument(err, "unable to decode image")
	}
	pi := PixurImageOld{
		SubImager: im.(SubImager),
	}
	if mime, err := schema.FromImageFormat(name); err != nil {
		return nil, status.InternalError(err, "Unknown format name", name)
	} else {
		pi.Mime = mime
	}
	if name == "gif" {
		if _, err := r.Seek(0, os.SEEK_SET); err != nil {
			return nil, status.InternalError(err, "unable to seek file")
		}
		g, err := gif.DecodeAll(r)
		if err != nil {
			return nil, status.InvalidArgument(err, "unable to decode gif")
		}
		// Ignore gifs that have only one frame
		if len(g.Delay) > 1 {
			dur, sts := GetGifDuration(g)
			if err != nil {
				return nil, sts
			}
			pi.AnimationInfo = &schema.AnimationInfo{
				Duration: dur,
			}
			// TODO: maybe skip the first second of frames like webm
		}
	}

	return &pi, nil
}

func FillImageConfig(f *os.File, p *schema.Pic) (image.Image, status.S) {
	if _, err := f.Seek(0, os.SEEK_SET); err != nil {
		return nil, status.InternalError(err, "unable to seek file")
	}
	defer f.Seek(0, os.SEEK_SET)

	im, imgType, err := image.Decode(f)
	if err == image.ErrFormat {
		// Try Webm
		im, sts := fillImageConfigFromWebm(f, p)
		if sts != nil {
			return nil, sts
		}
		return im, nil
	} else if err != nil {
		return nil, status.InvalidArgument(err, "unable to decode image")
	} else {
		// TODO: handle this error
		p.Mime, _ = schema.FromImageFormat(imgType)
		p.Width = int64(im.Bounds().Dx())
		p.Height = int64(im.Bounds().Dy())
	}

	if p.Mime == schema.Pic_GIF {
		if _, err := f.Seek(0, os.SEEK_SET); err != nil {
			return nil, status.InternalError(err, "unable to seek file")
		}

		GIF, err := gif.DecodeAll(f)
		if err != nil {
			return nil, status.InvalidArgument(err, "unable to decode gif")
		}
		// Ignore gifs that have only one frame
		if len(GIF.Delay) > 1 {
			dur, sts := GetGifDuration(GIF)
			if sts != nil {
				return nil, sts
			}
			p.AnimationInfo = &schema.AnimationInfo{
				Duration: dur,
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
func GetGifDuration(g *gif.GIF) (*durpb.Duration, status.S) {
	const maxFrameHundredths = 1 << 32
	var dur time.Duration
	// each delay unit is 1/100 of a second
	for _, frameHundredths := range g.Delay {
		fh64 := int64(frameHundredths)
		if fh64 > maxFrameHundredths || fh64 < 0 {
			return nil, status.InvalidArgumentf(nil, "GIF frame length exceeds max %v", frameHundredths)
		}
		// can't overflow
		framedur := 10 * time.Millisecond * time.Duration(fh64)
		if dur+framedur < 0 {
			return nil, status.InvalidArgumentf(nil, "GIF length exceeds max %v+%v", dur, framedur)
		}
		dur += framedur
	}

	return ptypes.DurationProto(dur), nil
}

type SubImager interface {
	image.Image
	SubImage(image.Rectangle) image.Image
}

// TODO: interpret image rotation metadata
func MakeThumbnail(img image.Image) image.Image {
	bounds := findMaxSquare(img.Bounds())

	var largeSquareImage image.Image

	if subImg, ok := img.(SubImager); ok {
		largeSquareImage = subImg.SubImage(bounds)
	} else {
		// this should be really rare.  Rather than panic if not a SubImager,
		// just use the slowpath.
		log.Printf("Warning, image is not a subimager %T", img)
		largeSquareImage = image.NewNRGBA(bounds)
		draw.Draw(largeSquareImage.(draw.Image), bounds, img, bounds.Min, draw.Src)
	}

	return resize.Resize(DefaultThumbnailWidth, DefaultThumbnailHeight, largeSquareImage,
		resize.Lanczos2)
}

func OutputThumbnail(im image.Image, mime schema.Pic_Mime, f *os.File) status.S {
	thumb := MakeThumbnail(im)
	switch mime {
	case schema.Pic_JPEG:
		if err := jpeg.Encode(f, thumb, nil); err != nil {
			return status.InternalError(err, "can't encode jpeg thumbnail")
		}
	case schema.Pic_GIF:
		if err := png.Encode(f, thumb); err != nil {
			return status.InternalError(err, "can't encode png thumbnail for gif")
		}
	case schema.Pic_PNG:
		if err := png.Encode(f, thumb); err != nil {
			return status.InternalError(err, "can't encode png thumbnail")
		}
	case schema.Pic_WEBM:
		if err := jpeg.Encode(f, thumb, nil); err != nil {
			return status.InternalError(err, "can't encode jpg thumbnail for webm")
		}
	default:
		return status.InternalErrorf(nil, "Unknown mime type %v", mime)
	}
	return nil
}

// TODO: remove this function
func SaveThumbnail(im image.Image, p *schema.Pic, pixPath string) status.S {
	path, sts := schema.PicFileThumbnailPath(pixPath, p.PicId, 0, schema.Pic_File_JPEG)
	if sts != nil {
		return sts
	}
	f, err := os.Create(path)
	if err != nil {
		return status.InternalError(err, "unable to create thumbnail")
	}
	defer f.Close()
	if err := jpeg.Encode(f, im, nil); err != nil {
		return status.InternalError(err, "unable to save jpg thumbnail")
	}
	return nil
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

func fillImageConfigFromWebm(tempFile *os.File, p *schema.Pic) (image.Image, status.S) {
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
				Duration: ptypes.DurationProto(dur),
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

func GetWebmConfig(filepath string) (*FFprobeConfig, status.S) {
	cmd := exec.Command("ffprobe",
		"-print_format", "json",
		"-v", "quiet", // disable version info
		"-show_format",
		"-show_streams",
		filepath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, status.InternalError(err, "unable to get stdout pipe")
	}
	if err := cmd.Start(); err != nil {
		return nil, status.InternalError(err, "unable to start ffprobe")
	}
	config := new(FFprobeConfig)
	if err := json.NewDecoder(stdout).Decode(config); err != nil {
		return nil, status.InternalError(err, "unable to decode ffprobe json")
	}
	if err := cmd.Wait(); err != nil {
		return nil, status.InternalError(err, "failed waiting for ffprobe to finish")
	}

	if config.Format.FormatName != "matroska,webm" {
		return nil, status.InvalidArgument(nil, "Only webm supported", config)
	}
	if config.Format.StreamCount == 0 {
		return nil, status.InvalidArgument(nil, "No Streams found", config)
	}
	// TODO: delete this whoel function, I don't care about it with ffmpeg.go in the picture.
	//if config.Format.Duration < 0 || config.Format.Duration > maxWebmDuration {
	//	return nil, status.InvalidArgument(nil, "Invalid Duration", config)
	//}

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
		return nil, status.InvalidArgument(nil, "Bad Video", config)
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
func getFirstWebmFrame(filepath string) (image.Image, status.S) {
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
		return nil, status.InternalError(err, "unable to create stdout pipe")
	}
	defer stdout.Close()

	if err := cmd.Start(); err != nil {
		return nil, status.InternalError(err, "unable to start ffmpeg")
	}
	defer cmd.Process.Kill()

	im, sts := keepLastImage(stdout)
	if sts != nil {
		return nil, sts
	}

	if err := cmd.Wait(); err != nil {
		return nil, status.InternalError(err, "unable to wait on ffmpeg")
	}

	return im, nil
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
			break
		} else if err != nil {
			return nil, status.InvalidArgument(err, "unable to decode png image")
		}
		im = lastIm
	}

	if im == nil {
		return nil, status.InvalidArgument(nil, "No frames in webm")
	}

	return im, nil
}
