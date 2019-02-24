package handlers

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"

	"pixur.org/pixur/api"
)

func init() {
	defaultPwtCoder = &pwtCoder{
		now:    time.Now,
		secret: []byte("secret"),
	}

	notafter, _ := ptypes.TimestampProto(time.Now().Add(authPwtDuration))
	notbefore, _ := ptypes.TimestampProto(time.Now().Add(-1 * time.Minute))
	testAuthPayload = &api.PwtPayload{
		Subject:   "0",
		NotAfter:  notafter,
		NotBefore: notbefore,
		Type:      api.PwtPayload_AUTH,
		TokenId:   1,
	}

	authToken, err := defaultPwtCoder.encode(testAuthPayload)
	if err != nil {
		panic(err)
	}
	testAuthToken = string(authToken)
}

var (
	testAuthSubject = int64(0)
	testAuthPayload *api.PwtPayload
	testAuthToken   string
)

type testClient struct {
	HTTPClient   *http.Client
	AuthOverride *api.PwtPayload
	DisableAuth  bool
}

func (c *testClient) Do(req *http.Request) (*http.Response, error) {
	if !c.DisableAuth {
		// Add in Auth
		var payload *api.PwtPayload
		if c.AuthOverride == nil {
			notafter, _ := ptypes.TimestampProto(time.Now().Add(authPwtDuration))
			notbefore, _ := ptypes.TimestampProto(time.Now().Add(-1 * time.Minute))
			payload = &api.PwtPayload{
				Subject:   "0",
				NotAfter:  notafter,
				NotBefore: notbefore,
				Type:      api.PwtPayload_AUTH,
			}
		} else {
			payload = c.AuthOverride
		}

		authToken, err := defaultPwtCoder.encode(payload)
		if err != nil {
			panic(err)
		}
		req.AddCookie(&http.Cookie{
			Name:  authPwtHeaderKey,
			Value: string(authToken),
		})
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

func bodyToText(body io.Reader) string {
	text, _ := ioutil.ReadAll(body)
	return string(text)
}
