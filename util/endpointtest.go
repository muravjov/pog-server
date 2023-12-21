package util

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Endpoint struct {
	Method  string
	Path    string
	Data    interface{}
	Handler func(w http.ResponseWriter, r *http.Request)
	IsJSON  bool
}

func DoEndpoint(endpoint *Endpoint, t *testing.T) *http.Response {
	ser := httptest.NewServer(http.HandlerFunc(endpoint.Handler))
	defer ser.Close()

	getMethod := func(method string) string {
		if endpoint.Method != "" {
			return endpoint.Method
		}
		return method
	}

	var req *http.Request
	url := ser.URL + endpoint.Path
	if endpoint.IsJSON {
		if endpoint.Data != nil {
			b, err := json.Marshal(endpoint.Data)
			assert.NoError(t, err)

			req, _ = http.NewRequest(getMethod(http.MethodPost), url, bytes.NewReader(b))
			req.Header.Add("Content-Type", "application/json")
		} else {
			req, _ = http.NewRequest(getMethod(http.MethodGet), url, nil)
		}
	} else {
		req = NewFormRequestFromMap(url, endpoint.Data.(map[string]string))
	}

	res, err := ser.Client().Do(req)
	assert.NoError(t, err)

	return res
}

func InvokeEndpoint(endpoint *Endpoint, dumpJSON bool, t *testing.T) {
	res := DoEndpoint(endpoint, t)
	// nolint: errcheck
	defer res.Body.Close()
	assert.EqualValues(t, http.StatusOK, res.StatusCode)

	if dumpJSON {
		v := new(interface{})
		err := json.NewDecoder(res.Body).Decode(v)
		assert.NoError(t, err)

		DumpIndent(v)
	} else {
		if _, err := io.Copy(os.Stdout, res.Body); err != nil {
			assert.NoError(t, err)
		}
	}
}

func NewFormRequestFromMap(formUrl string, body map[string]string) *http.Request {
	dataStr := Map2URLPath(body)

	req, _ := http.NewRequest(http.MethodPost, formUrl, strings.NewReader(dataStr))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	return req
}

func Map2URLPath(m map[string]string) string {
	data := url.Values{}
	for k, v := range m {
		data.Set(k, v)
	}
	return data.Encode()
}

type EndpointList []struct {
	Endpoint   *Endpoint
	OnResponse func(*http.Response)
}

func (lst EndpointList) Do(t *testing.T) {
	for _, item := range lst {
		res := DoEndpoint(item.Endpoint, t)
		item.OnResponse(res)
	}
}
