//go:build !lambdahttpadapter.partial || (lambdahttpadapter.partial && lambdahttpadapter.apigwv2)

package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"github.com/aws/aws-lambda-go/events"
	"io"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"
)

func convertApiGwV2Request(ctx context.Context, event events.APIGatewayV2HTTPRequest) (*http.Request, error) {
	url := buildFullRequestURL(event.RequestContext.DomainName, event.RawPath, event.RequestContext.HTTP.Path, buildQuery(event.RawQueryString, event.QueryStringParameters))
	req, err := http.NewRequestWithContext(ctx, event.RequestContext.HTTP.Method, url, getBody(event.Body, event.IsBase64Encoded))
	if err != nil {
		return nil, err
	}

	if event.Cookies != nil {
		for _, v := range event.Cookies {
			req.Header.Add("Cookie", v)
		}
	}

	for k, v := range event.Headers {
		req.Header.Add(k, v)
	}

	req.Proto = event.RequestContext.HTTP.Protocol
	pMajor, pMinor, ok := http.ParseHTTPVersion(req.Proto)
	if ok {
		req.ProtoMajor, req.ProtoMinor = pMajor, pMinor
	}

	req.RemoteAddr = buildRemoteAddr(event.RequestContext.HTTP.SourceIP)
	req.RequestURI = req.URL.RequestURI()

	return req, nil
}

type apiGwV2ResponseWriter struct {
	headersWritten   bool
	contentTypeSet   bool
	contentLengthSet bool
	headers          http.Header
	body             bytes.Buffer
	res              events.APIGatewayV2HTTPResponse
}

func (w *apiGwV2ResponseWriter) Header() http.Header {
	return w.headers
}

func (w *apiGwV2ResponseWriter) Write(p []byte) (int, error) {
	w.WriteHeader(http.StatusOK)
	return w.body.Write(p)
}

func (w *apiGwV2ResponseWriter) WriteHeader(statusCode int) {
	if !w.headersWritten {
		w.headersWritten = true
		w.res.StatusCode = statusCode

		for k, values := range w.headers {
			if strings.EqualFold("set-cookie", k) {
				w.res.Cookies = values
			} else {
				if len(values) == 0 {
					w.res.Headers[k] = ""
				} else if len(values) == 1 {
					w.res.Headers[k] = values[0]
				} else {
					if w.res.MultiValueHeaders == nil {
						w.res.MultiValueHeaders = make(map[string][]string)
					}

					w.res.MultiValueHeaders[k] = values
				}
			}
		}
	}
}

func handleApiGwV2(ctx context.Context, event events.APIGatewayV2HTTPRequest, adapter AdapterFunc) (events.APIGatewayV2HTTPResponse, error) {
	req, err := convertApiGwV2Request(ctx, event)
	if err != nil {
		var def events.APIGatewayV2HTTPResponse
		return def, err
	}

	w := apiGwV2ResponseWriter{
		headers: make(http.Header),
		res: events.APIGatewayV2HTTPResponse{
			Headers: make(map[string]string),
			Cookies: make([]string, 0),
		},
	}

	if err = adapter(ctx, req, &w); err != nil {
		var def events.APIGatewayV2HTTPResponse
		return def, err
	}

	b, err := io.ReadAll(&w.body)
	if err != nil {
		var def events.APIGatewayV2HTTPResponse
		return def, err
	}

	if !w.contentTypeSet {
		w.res.Headers["Content-Type"] = http.DetectContentType(b)
	}

	if !w.contentLengthSet {
		w.res.Headers["Content-Length"] = strconv.Itoa(len(b))
	}

	if utf8.Valid(b) {
		w.res.Body = string(b)
	} else {
		w.res.IsBase64Encoded = true
		w.res.Body = base64.StdEncoding.EncodeToString(b)
	}

	return w.res, nil
}

func NewAPIGatewayV2Handler(adapter AdapterFunc) func(context.Context, events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	return NewHandler(handleApiGwV2, adapter)
}
