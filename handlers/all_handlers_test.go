package handlers

import (
	"io"
	"net/http"
	"net/url"
	"strings"
)

type testClient struct {
	HTTPClient  *http.Client
	DisableXSRF bool
}

func (c *testClient) Do(req *http.Request) (*http.Response, error) {
	if !c.DisableXSRF {
		b64XsrfToken := make([]byte, b64XsrfTokenLength)
		b64XsrfEnc.Encode(b64XsrfToken, make([]byte, xsrfTokenLength))
		req.AddCookie(&http.Cookie{
			Name:  xsrfCookieName,
			Value: string(b64XsrfToken),
		})
		req.Header.Add(xsrfHeaderName, string(b64XsrfToken))
	}

	var httpClient *http.Client
	if c.HTTPClient != nil {
		httpClient = c.HTTPClient
	} else {
		httpClient = http.DefaultClient
	}
	return httpClient.Do(req)
}

func (c *testClient) Post(url string, bodyType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", bodyType)
	return c.Do(req)
}

func (c *testClient) PostForm(url string, data url.Values) (*http.Response, error) {
	return c.Post(url, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
}

func (c *testClient) Get(url string) (resp *http.Response, err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}
