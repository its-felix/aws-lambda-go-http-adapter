package handler

import (
	"context"
	"encoding/base64"
	"github.com/aws/aws-lambda-go/events"
	"net/http"
	"net/url"
	"unicode/utf8"
)

func apiGwV1RequestConverter(ctx context.Context, event events.APIGatewayProxyRequest) (*http.Request, error) {
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

	req.RemoteAddr = event.RequestContext.Identity.SourceIP + ":http"
	req.RequestURI = req.URL.RequestURI()

	return req, nil
}

func apiGwV1ResponseInitializer(ctx context.Context) *ResponseWriterProxy {
	return NewResponseWriterProxy()
}

func apiGwV1ResponseFinalizer(ctx context.Context, w *ResponseWriterProxy) (events.APIGatewayProxyResponse, error) {
	out := events.APIGatewayProxyResponse{
		StatusCode: w.Status,
		Headers:    make(map[string]string),
	}

	for k, values := range w.Headers {
		if len(values) == 0 {
			out.Headers[k] = ""
		} else if len(values) == 1 {
			out.Headers[k] = values[0]
		} else {
			if out.MultiValueHeaders == nil {
				out.MultiValueHeaders = make(map[string][]string)
			}

			out.MultiValueHeaders[k] = values
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

func NewAPIGatewayV1Handler(adapter AdapterFunc) func(context.Context, events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return NewHandler(apiGwV1RequestConverter, apiGwV1ResponseInitializer, apiGwV1ResponseFinalizer, adapter)
}
