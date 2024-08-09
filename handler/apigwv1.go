//go:build !lambdahttpadapter.partial || (lambdahttpadapter.partial && lambdahttpadapter.apigwv1)

package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"github.com/aws/aws-lambda-go/events"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"unicode/utf8"
)

func convertApiGwV1Request(ctx context.Context, event events.APIGatewayProxyRequest) (*http.Request, error) {
	q := make(url.Values)

	if len(event.MultiValueQueryStringParameters) > 0 {
		for k, values := range event.MultiValueQueryStringParameters {
			for _, v := range values {
				q.Add(k, v)
			}
		}
	} else if len(event.QueryStringParameters) > 0 {
		for k, v := range event.QueryStringParameters {
			q.Add(k, v)
		}
	}

	rUrl := buildFullRequestURL(event.RequestContext.DomainName, event.Path, event.RequestContext.Path, q.Encode())
	req, err := http.NewRequestWithContext(ctx, event.HTTPMethod, rUrl, getBody(event.Body, event.IsBase64Encoded))
	if err != nil {
		return nil, err
	}

	if event.MultiValueHeaders != nil {
		for k, values := range event.MultiValueHeaders {
			for _, v := range values {
				req.Header.Add(k, v)
			}
		}
	} else {
		for k, v := range event.Headers {
			req.Header.Add(k, v)
		}
	}

	req.Proto = event.RequestContext.Protocol
	pMajor, pMinor, ok := http.ParseHTTPVersion(req.Proto)
	if ok {
		req.ProtoMajor, req.ProtoMinor = pMajor, pMinor
	}

	req.RemoteAddr = buildRemoteAddr(event.RequestContext.Identity.SourceIP)
	req.RequestURI = req.URL.RequestURI()

	return req, nil
}

type apiGwV1ResponseWriter struct {
	headersWritten   bool
	contentTypeSet   bool
	contentLengthSet bool
	headers          http.Header
	body             bytes.Buffer
	res              events.APIGatewayProxyResponse
}

func (w *apiGwV1ResponseWriter) Header() http.Header {
	return w.headers
}

func (w *apiGwV1ResponseWriter) Write(p []byte) (int, error) {
	w.WriteHeader(http.StatusOK)
	return w.body.Write(p)
}

func (w *apiGwV1ResponseWriter) WriteHeader(statusCode int) {
	if !w.headersWritten {
		w.headersWritten = true
		w.res.StatusCode = statusCode

		for k, values := range w.headers {
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

func handleApiGwV1(ctx context.Context, event events.APIGatewayProxyRequest, adapter AdapterFunc) (events.APIGatewayProxyResponse, error) {
	req, err := convertApiGwV1Request(ctx, event)
	if err != nil {
		var def events.APIGatewayProxyResponse
		return def, err
	}

	w := apiGwV1ResponseWriter{
		headers: make(http.Header),
		res: events.APIGatewayProxyResponse{
			Headers: make(map[string]string),
		},
	}

	if err = adapter(ctx, req, &w); err != nil {
		var def events.APIGatewayProxyResponse
		return def, err
	}

	b, err := io.ReadAll(&w.body)
	if err != nil {
		var def events.APIGatewayProxyResponse
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

func NewAPIGatewayV1Handler(adapter AdapterFunc) func(context.Context, events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return NewHandler(handleApiGwV1, adapter)
}
