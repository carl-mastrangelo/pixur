package imaging

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
	"time"
)

func TestCheckValidVideo_BadFormat(t *testing.T) {
	resp := probeResponse{
		Format: probeFormat{
			FormatName: "exe",
		},
	}
	if _, err := checkValidVideo(&resp); err == nil {
		t.Fatal("expected an error")
	}
}

func TestCheckValidVideo_BadStreamCount(t *testing.T) {
	resp := probeResponse{
		Format: probeFormat{
			FormatName:  "matroska,webm",
			StreamCount: 0,
		},
	}
	if _, err := checkValidVideo(&resp); err == nil {
		t.Fatal("expected an error")
	}
}

func TestCheckValidVideo_BadDuration(t *testing.T) {
	resp := probeResponse{
		Format: probeFormat{
			FormatName:  "matroska,webm",
			StreamCount: 1,
			Duration:    "",
		},
	}
	if _, err := checkValidVideo(&resp); err == nil {
		t.Fatal("expected an error")
	}
}

func TestCheckValidVideo_LongDuration(t *testing.T) {
	resp := probeResponse{
		Format: probeFormat{
			FormatName:  "matroska,webm",
			StreamCount: 1,
			Duration:    "1000.0",
		},
	}
	if _, err := checkValidVideo(&resp); err == nil {
		t.Fatal("expected an error")
	}
}

func TestCheckValidVideo_BadVideoStream(t *testing.T) {
	resp := probeResponse{
		Format: probeFormat{
			FormatName:  "matroska,webm",
			StreamCount: 1,
			Duration:    "10.0",
		},
		Streams: []probeStream{{
			CodecType: "video",
			CodecName: "h264",
		}},
	}
	if _, err := checkValidVideo(&resp); err == nil {
		t.Fatal("expected an error")
	}
}

func TestCheckValidVideo_BadAudioStream(t *testing.T) {
	resp := probeResponse{
		Format: probeFormat{
			FormatName:  "matroska,webm",
			StreamCount: 1,
			Duration:    "10.0",
		},
		Streams: []probeStream{{
			CodecType: "audio",
			CodecName: "mp3",
		}},
	}
	if _, err := checkValidVideo(&resp); err == nil {
		t.Fatal("expected an error")
	}
}

func TestCheckValidVideo_NoVideoStream(t *testing.T) {
	resp := probeResponse{
		Format: probeFormat{
			FormatName:  "matroska,webm",
			StreamCount: 1,
			Duration:    "10.0",
		},
		Streams: []probeStream{{
			CodecType: "audio",
			CodecName: "vorbis",
		}},
	}
	if _, err := checkValidVideo(&resp); err == nil {
		t.Fatal("expected an error")
	}
}

func TestCheckValidVideo_MultipleVideoStream(t *testing.T) {
	resp := probeResponse{
		Format: probeFormat{
			FormatName:  "matroska,webm",
			StreamCount: 2,
			Duration:    "10.0",
		},
		Streams: []probeStream{{
			CodecType: "video",
			CodecName: "vp8",
		}, {
			CodecType: "video",
			CodecName: "vp9",
		}},
	}
	if _, err := checkValidVideo(&resp); err != nil {
		t.Fatal(err)
	}
}

func TestCheckValidVideo_VideoAndAudio_Webm(t *testing.T) {
	resp := probeResponse{
		Format: probeFormat{
			FormatName:  "matroska,webm",
			StreamCount: 2,
			Duration:    "10.0",
		},
		Streams: []probeStream{{
			CodecType: "video",
			CodecName: "vp9",
		}, {
			CodecType: "audio",
			CodecName: "opus",
		}},
	}
	if format, err := checkValidVideo(&resp); err != nil {
		t.Fatal(err)
	} else if ImageFormat(format) != DefaultWebmFormat {
		t.Fatal("bad format", format, DefaultWebmFormat)
	}
}

func TestCheckValidVideo_VideoAndAudio_Mp4(t *testing.T) {
	resp := probeResponse{
		Format: probeFormat{
			FormatName:  "mov,mp4,m4a,3gp,3g2,mj2",
			StreamCount: 2,
			Duration:    "10.0",
		},
		Streams: []probeStream{{
			CodecType: "video",
			CodecName: "h264",
		}, {
			CodecType: "audio",
			CodecName: "aac",
		}},
	}
	if format, err := checkValidVideo(&resp); err != nil {
		t.Fatal(err)
	} else if ImageFormat(format) != DefaultMp4Format {
		t.Fatal("bad format", format, DefaultMp4Format)
	}
}

func TestParseDuration(t *testing.T) {
	if d, err := parseFfmpegDuration("1.1.1"); err == nil {
		t.Fatal("Expected failure, but parsed", d)
	}
	if d, err := parseFfmpegDuration("1"); err == nil {
		t.Fatal("Expected failure, but parsed", d)
	}
	if d, err := parseFfmpegDuration(".1"); err == nil {
		t.Fatal("Expected failure, but parsed", d)
	}
	if d, err := parseFfmpegDuration("1.9e8"); err == nil {
		t.Fatal("Expected failure, but parsed", d)
	}
	if d, err := parseFfmpegDuration("A.8"); err == nil {
		t.Fatal("Expected failure, but parsed", d)
	}
	d, err := parseFfmpegDuration("123.456789")
	if err != nil {
		t.Fatal(err)
	}
	if d != time.Duration(123456789000) {
		t.Fatal("time mismatch", d, time.Duration(123456789000))
	}
	d, err = parseFfmpegDuration("123.123456789999999")
	if err != nil {
		t.Fatal(err)
	}
	if expected := time.Duration(123123456789); d != expected {
		t.Fatal("time mismatch", d, expected)
	}
}

func TestKeepLastImage(t *testing.T) {
	var buf = new(bytes.Buffer)
	im := image.NewNRGBA(image.Rect(0, 0, 5, 10))
	if err := png.Encode(buf, im); err != nil {
		t.Fatal(err)
	}

	im.Set(0, 0, color.White)
	if err := png.Encode(buf, im); err != nil {
		t.Fatal(err)
	}

	out, err := keepLastImage(buf)
	if err != nil {
		t.Fatal(err)
	}
	if out.Bounds() != im.Bounds() {
		t.Fatal("Wrong bounds", out.Bounds())
	}
	if out.(*image.NRGBA).NRGBAAt(0, 0) != im.NRGBAAt(0, 0) {
		t.Fatal("not last image")
	}
}

func TestKeepLastImageBadLastImage(t *testing.T) {
	var buf = new(bytes.Buffer)
	im := image.NewNRGBA(image.Rect(0, 0, 5, 10))
	if err := png.Encode(buf, im); err != nil {
		t.Fatal(err)
	}

	if _, err := buf.WriteString("XXXXXXXXXXXXXXXX"); err != nil {
		t.Fatal(err)
	}

	if _, err := keepLastImage(buf); err == nil {
		t.Fatal("Expected an error")
	}
}

func TestKeepLastImageNoData(t *testing.T) {
	var buf = new(bytes.Buffer)

	if _, err := keepLastImage(buf); err == nil {
		t.Fatal("Expected an error")
	}
}

func TestKeepLastImageTooManyImages(t *testing.T) {
	var buf = new(bytes.Buffer)
	im := image.NewNRGBA(image.Rect(0, 0, 5, 5))
	for i := 0; i < 500; i++ {
		if err := png.Encode(buf, im); err != nil {
			t.Fatal(err)
		}
	}

	im.Set(0, 0, color.White)

	if err := png.Encode(buf, im); err != nil {
		t.Fatal(err)
	}

	out, err := keepLastImage(buf)
	if err != nil {
		t.Fatal(err)
	}
	if out.Bounds() != im.Bounds() {
		t.Fatal("Wrong bounds", out.Bounds())
	}
	if out.(*image.NRGBA).NRGBAAt(0, 0) == im.NRGBAAt(0, 0) {
		t.Fatal("should not be last image")
	}
}
