package handlers

import (
	"crypto/rand"
	"crypto/rsa"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func init() {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	jwtDec = &jwtDecoder{
		key: &key.PublicKey,
	}
	jwtEnc = &jwtEncoder{
		key: key,
	}
}

type testClient struct {
	HTTPClient  *http.Client
	DisableXSRF bool
	JwtOverride *JwtPayload
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
	var payload *JwtPayload
	if c.JwtOverride == nil {

		payload = &JwtPayload{
			Subject:    "0",
			Expiration: time.Now().Add(jwtLifetime).Unix(),
			NotBefore:  time.Now().Add(-1 * time.Minute).Unix(),
		}
	} else {
		payload = c.JwtOverride
	}

	jwt, err := jwtEnc.Sign(payload)
	if err != nil {
		panic(err)
	}
	req.AddCookie(&http.Cookie{
		Name:  jwtCookieName,
		Value: string(jwt),
	})

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
