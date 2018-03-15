package handlers

import (
	"net/url"
	"strings"
	"testing"

	"pixur.org/pixur/api"
)

func TestParseCapsChange(t *testing.T) {
	var p params

	vals := url.Values{}
	vals.Set("dummy0", p.True())
	vals.Set("dummy1", p.False())

	yes, no, err := p.parseCapsChange("dummy", vals)
	if err != nil {
		t.Fatal("unexpected error", err)
	}
	if len(yes) != 1 || yes[0] != api.Capability_Cap(0) {
		t.Error("bad yes value", yes)
	}
	if len(no) != 1 || no[0] != api.Capability_Cap(1) {
		t.Error("bad no value", no)
	}
}

func TestParseCapsChange_FailsOnBadCap(t *testing.T) {
	var p params

	vals := url.Values{}
	vals.Set("dummy50000000", p.True())

	_, _, err := p.parseCapsChange("dummy", vals)
	if !strings.Contains(err.Error(), "unknown cap") {
		t.Fatal("wrong error", err)
	}
}

func TestParseCapsChange_FailsOnBadInt(t *testing.T) {
	var p params

	vals := url.Values{}
	vals.Set("dummy_letter", p.True())

	_, _, err := p.parseCapsChange("dummy", vals)
	if !strings.Contains(err.Error(), "can't parse") {
		t.Fatal("wrong error", err)
	}
}

func TestParseCapsChange_FailsOnBadValue(t *testing.T) {
	var p params

	vals := url.Values{}
	vals.Set("dummy1", "bogus")

	_, _, err := p.parseCapsChange("dummy", vals)
	if !strings.Contains(err.Error(), "unknown value") {
		t.Fatal("wrong error", err)
	}
}

func TestParseCapsChange_FailsOnNoValue(t *testing.T) {
	var p params

	vals := url.Values{}
	vals["dummy1"] = nil

	_, _, err := p.parseCapsChange("dummy", vals)
	if !strings.Contains(err.Error(), "bad value(s)") {
		t.Fatal("wrong error", err)
	}
}

func TestParseCapsChange_FailsOnMultiValue(t *testing.T) {
	var p params

	vals := url.Values{}
	vals.Add("dummy1", p.True())
	vals.Add("dummy1", p.True())

	_, _, err := p.parseCapsChange("dummy", vals)
	if !strings.Contains(err.Error(), "bad value(s)") {
		t.Fatal("wrong error", err)
	}
}

func TestParseCapsChange_IgnoresOther(t *testing.T) {
	var p params

	vals := url.Values{}
	vals.Set("dummy_letter", p.True())

	yes, no, err := p.parseCapsChange("cap", vals)
	if err != nil {
		t.Fatal("unexpected error", err)
	}
	if len(yes) != 0 {
		t.Error("bad yes value", yes)
	}
	if len(no) != 0 {
		t.Error("bad no value", no)
	}
}
