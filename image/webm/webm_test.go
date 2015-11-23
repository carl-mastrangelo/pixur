package webm

import (
	"testing"
	"time"
)

func TestCheckValidWebm_BadFormat(t *testing.T) {
	resp := probeResponse{
		Format: probeFormat{
			FormatName: "mp4",
		},
	}
	if err := checkValidWebm(resp); err == nil {
		t.Fatal("expected an error")
	}
}

func TestCheckValidWebm_BadStreamCount(t *testing.T) {
	resp := probeResponse{
		Format: probeFormat{
			FormatName:  "matroska,webm",
			StreamCount: 0,
		},
	}
	if err := checkValidWebm(resp); err == nil {
		t.Fatal("expected an error")
	}
}

func TestCheckValidWebm_BadDuration(t *testing.T) {
	resp := probeResponse{
		Format: probeFormat{
			FormatName:  "matroska,webm",
			StreamCount: 1,
			Duration:    "",
		},
	}
	if err := checkValidWebm(resp); err == nil {
		t.Fatal("expected an error")
	}
}

func TestCheckValidWebm_LongDuration(t *testing.T) {
	resp := probeResponse{
		Format: probeFormat{
			FormatName:  "matroska,webm",
			StreamCount: 1,
			Duration:    "1000.0",
		},
	}
	if err := checkValidWebm(resp); err == nil {
		t.Fatal("expected an error")
	}
}

func TestCheckValidWebm_BadVideoStream(t *testing.T) {
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
	if err := checkValidWebm(resp); err == nil {
		t.Fatal("expected an error")
	}
}

func TestCheckValidWebm_BadAudioStream(t *testing.T) {
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
	if err := checkValidWebm(resp); err == nil {
		t.Fatal("expected an error")
	}
}

func TestCheckValidWebm_NoVideoStream(t *testing.T) {
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
	if err := checkValidWebm(resp); err == nil {
		t.Fatal("expected an error")
	}
}

func TestCheckValidWebm_MultipleVideoStream(t *testing.T) {
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
	if err := checkValidWebm(resp); err != nil {
		t.Fatal(err)
	}
}

func TestCheckValidWebm_VideoAndAudio(t *testing.T) {
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
	if err := checkValidWebm(resp); err != nil {
		t.Fatal(err)
	}
}

func TestParseDuration(t *testing.T) {
	if d, err := parseDuration("1.1.1"); err == nil {
		t.Fatal("Expected failure, but parsed", d)
	}
	if d, err := parseDuration("1"); err == nil {
		t.Fatal("Expected failure, but parsed", d)
	}
	if d, err := parseDuration(".1"); err == nil {
		t.Fatal("Expected failure, but parsed", d)
	}
	if d, err := parseDuration("1."); err == nil {
		t.Fatal("Expected failure, but parsed", d)
	}
	if d, err := parseDuration("1.9e8"); err == nil {
		t.Fatal("Expected failure, but parsed", d)
	}
	if d, err := parseDuration("A.8"); err == nil {
		t.Fatal("Expected failure, but parsed", d)
	}
	d, err := parseDuration("123.456789")
	if err != nil {
		t.Fatal(err)
	}
	if d != time.Duration(123456789000) {
		t.Fatal("time mismatch", d, time.Duration(123456789000))
	}
}
