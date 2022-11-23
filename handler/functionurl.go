package handler

import (
	"context"
	"encoding/base64"
	"github.com/aws/aws-lambda-go/events"
	"net/http"
	"strings"
	"unicode/utf8"
)

func functionURLRequestConverter(ctx context.Context, event events.LambdaFunctionURLRequest) (*http.Request, error) {
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

	for k, valuesRaw := range event.Headers {
		for _, v := range strings.Split(valuesRaw, ",") {
			req.Header.Add(k, v)
		}
	}

	req.Proto = event.RequestContext.HTTP.Protocol
	pMajor, pMinor, ok := http.ParseHTTPVersion(req.Proto)
	if ok {
		req.ProtoMajor, req.ProtoMinor = pMajor, pMinor
	}

	req.RemoteAddr = event.RequestContext.HTTP.SourceIP + ":http"
	req.RequestURI = req.URL.RequestURI()

	return req, nil
}

func functionURLResponseInitializer(ctx context.Context) *ResponseWriterProxy {
	return NewResponseWriterProxy()
}

func functionURLResponseFinalizer(ctx context.Context, w *ResponseWriterProxy) (events.LambdaFunctionURLResponse, error) {
	out := events.LambdaFunctionURLResponse{
		StatusCode: w.Status,
		Headers:    make(map[string]string),
		Cookies:    make([]string, 0),
	}

	for k, values := range w.Headers {
		if strings.EqualFold("set-cookie", k) {
			out.Cookies = values
		} else {
			if len(values) == 0 {
				out.Headers[k] = ""
			} else if len(values) == 1 {
				out.Headers[k] = values[0]
			} else {
				out.Headers[k] = strings.Join(values, ",")
			}
		}
	}

	b := w.Body.Bytes()
	if utf8.Valid(b) {
		out.Body = string(b)
	} else {
		out.IsBase64Encoded = true
		out.Body = base64.StdEncoding.EncodeToString(b)
	}

	return out, nil
}

func NewFunctionURLHandler(adapter AdapterFunc) func(context.Context, events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	return NewHandler(functionURLRequestConverter, functionURLResponseInitializer, functionURLResponseFinalizer, adapter)
}
