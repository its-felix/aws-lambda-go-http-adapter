package handler

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func buildQuery(rawQuery string, queryParams map[string]string) string {
	if rawQuery != "" {
		return rawQuery
	} else if len(queryParams) > 0 {
		q := make(url.Values)

		for k, v := range queryParams {
			q.Add(k, v)
		}

		return "?" + q.Encode()
	}

	return ""
}

func buildFullRequestURL(host string, path string, altPath string, query string) string {
	rUrl := path

	if rUrl == "" {
		rUrl = altPath
	}

	if !strings.HasPrefix(rUrl, "/") {
		rUrl = "/" + rUrl
	}

	rUrl = "https://" + host + rUrl

	if query != "" {
		rUrl += "?" + query
	}

	return rUrl
}

func visitHeader(visitor func(string, []string), h map[string]string, cookies []string) {
	visitor("Cookie", cookies)

	for k, values := range h {
		visitor(k, strings.Split(values, ","))
	}
}

func convertBody(body string, isB64 bool) ([]byte, error) {
	var r io.Reader
	r = strings.NewReader(body)

	if isB64 {
		r = base64.NewDecoder(base64.StdEncoding, r)
	}

	return io.ReadAll(r)
}

func getBody(body string, isB64 bool) io.Reader {
	if body == "" {
		return nil
	}

	var b io.Reader
	b = strings.NewReader(body)

	if isB64 {
		b = base64.NewDecoder(base64.StdEncoding, b)
	}

	return b
}

type ResponseWriterProxy struct {
	Status  int
	Headers http.Header
	Body    bytes.Buffer
}

func (w *ResponseWriterProxy) Header() http.Header {
	return w.Headers
}

func (w *ResponseWriterProxy) Write(p []byte) (int, error) {
	return w.Body.Write(p)
}

func (w *ResponseWriterProxy) WriteHeader(statusCode int) {
	w.Status = statusCode
}

func NewResponseWriterProxy() *ResponseWriterProxy {
	return &ResponseWriterProxy{
		Status:  http.StatusOK,
		Headers: make(http.Header),
	}
}
