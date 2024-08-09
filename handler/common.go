package handler

import (
	"encoding/base64"
	"io"
	"net"
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

func buildRemoteAddr(sourceIp string) string {
	return net.JoinHostPort(sourceIp, "http")
}
