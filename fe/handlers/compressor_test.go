package handlers

import (
	"compress/gzip"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCompressionHandlerNoAccept(t *testing.T) {
	hh := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hi"))
	})

	s := httptest.NewServer(&compressionHandler{next: hh})
	defer s.Close()

	resp, err := s.Client().Get(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Error("not ok", resp.StatusCode)
	}
	if have, want := resp.Header.Get("Content-Encoding"), ""; have != want {
		t.Error("have", have, "want", want)
	}
}

func TestCompressionHandlerWrongAccept(t *testing.T) {
	hh := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hi"))
	})

	s := httptest.NewServer(&compressionHandler{next: hh})
	defer s.Close()

	req, err := http.NewRequest(http.MethodGet, s.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept-Encoding", "fzip")

	resp, err := s.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Error("not ok", resp.StatusCode)
	}
	if have, want := resp.Header.Get("Content-Encoding"), ""; have != want {
		t.Error("have", have, "want", want)
	}
}

func TestCompressionHandlerAccept(t *testing.T) {
	hh := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hi"))
	})

	s := httptest.NewServer(&compressionHandler{next: hh})
	defer s.Close()

	req, err := http.NewRequest(http.MethodGet, s.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := s.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Error("not ok", resp.StatusCode)
	}
	if have, want := resp.Header.Get("Content-Encoding"), "gzip"; have != want {
		t.Error("have", have, "want", want)
	}

	gr, err := gzip.NewReader(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer gr.Close()

	dst, err := ioutil.ReadAll(gr)
	if err != nil {
		t.Fatal(err)
	}

	if string(dst) != "hi" {
		t.Error(dst)
	}
}

func TestCompressionHandlerMultiAccept(t *testing.T) {
	hh := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hi"))
	})

	s := httptest.NewServer(&compressionHandler{next: hh})
	defer s.Close()

	req, err := http.NewRequest(http.MethodGet, s.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept-Encoding", "fzip, gzip")

	resp, err := s.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Error("not ok", resp.StatusCode)
	}
	if have, want := resp.Header.Get("Content-Encoding"), "gzip"; have != want {
		t.Error("have", have, "want", want)
	}

	gr, err := gzip.NewReader(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer gr.Close()

	dst, err := ioutil.ReadAll(gr)
	if err != nil {
		t.Fatal(err)
	}

	if string(dst) != "hi" {
		t.Error(dst)
	}
}

func TestCompressionHandlerOverride(t *testing.T) {
	hh := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "plain ol text")
		w.Write([]byte("hi"))
	})

	s := httptest.NewServer(&compressionHandler{next: hh})
	defer s.Close()

	req, err := http.NewRequest(http.MethodGet, s.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept-Encoding", "fzip, gzip")

	resp, err := s.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Error("not ok", resp.StatusCode)
	}
	if have, want := resp.Header.Get("Content-Encoding"), "plain ol text"; have != want {
		t.Error("have", have, "want", want)
	}

	dst, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(dst) != "hi" {
		t.Error(dst)
	}
}
