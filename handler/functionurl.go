//go:build !lambdahttpadapter.partial || (lambdahttpadapter.partial && lambdahttpadapter.functionurl)

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
	"sync/atomic"
	"unicode/utf8"
)

func convertFunctionURLRequest(ctx context.Context, event events.LambdaFunctionURLRequest) (*http.Request, error) {
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

// region classic
type functionURLResponseWriter struct {
	headersWritten   bool
	contentTypeSet   bool
	contentLengthSet bool
	headers          http.Header
	body             bytes.Buffer
	res              events.LambdaFunctionURLResponse
}

func (w *functionURLResponseWriter) Header() http.Header {
	return w.headers
}

func (w *functionURLResponseWriter) Write(p []byte) (int, error) {
	w.WriteHeader(http.StatusOK)
	return w.body.Write(p)
}

func (w *functionURLResponseWriter) WriteHeader(statusCode int) {
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
					w.res.Headers[k] = strings.Join(values, ",")
				}
			}

			if strings.EqualFold("content-type", k) {
				w.contentTypeSet = true
			} else if strings.EqualFold("content-length", k) {
				w.contentLengthSet = true
			}
		}
	}
}

func handleFunctionURL(ctx context.Context, event events.LambdaFunctionURLRequest, adapter AdapterFunc) (events.LambdaFunctionURLResponse, error) {
	req, err := convertFunctionURLRequest(ctx, event)
	if err != nil {
		var def events.LambdaFunctionURLResponse
		return def, err
	}

	w := functionURLResponseWriter{
		headers: make(http.Header),
		res: events.LambdaFunctionURLResponse{
			Headers: make(map[string]string),
			Cookies: make([]string, 0),
		},
	}

	if err = adapter(ctx, req, &w); err != nil {
		var def events.LambdaFunctionURLResponse
		return def, err
	}

	b, err := io.ReadAll(&w.body)
	if err != nil {
		var def events.LambdaFunctionURLResponse
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

func NewFunctionURLHandler(adapter AdapterFunc) func(context.Context, events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	return NewHandler(handleFunctionURL, adapter)
}

// endregion

// region streaming
type functionURLStreamingResponseWriter struct {
	headers        http.Header
	headersWritten int32
	body           io.WriteCloser
	resCh          chan<- events.LambdaFunctionURLStreamingResponse
}

func (w *functionURLStreamingResponseWriter) Header() http.Header {
	return w.headers
}

func (w *functionURLStreamingResponseWriter) Write(p []byte) (int, error) {
	w.WriteHeader(http.StatusOK)
	return w.body.Write(p)
}

func (w *functionURLStreamingResponseWriter) WriteHeader(statusCode int) {
	if atomic.CompareAndSwapInt32(&w.headersWritten, 0, 1) {
		pr, pw := io.Pipe()
		w.body = pw

		headers := make(map[string]string)
		cookies := make([]string, 0)

		for k, values := range w.headers {
			if strings.EqualFold("set-cookie", k) {
				cookies = values
			} else {
				if len(values) == 0 {
					headers[k] = ""
				} else if len(values) == 1 {
					headers[k] = values[0]
				} else {
					headers[k] = strings.Join(values, ",")
				}
			}
		}

		w.resCh <- events.LambdaFunctionURLStreamingResponse{
			StatusCode: statusCode,
			Headers:    headers,
			Body:       pr,
			Cookies:    cookies,
		}
	}
}

func (w *functionURLStreamingResponseWriter) Close() error {
	if w.body == nil {
		return nil
	}

	return w.body.Close()
}

func handleFunctionURLStreaming(ctx context.Context, event events.LambdaFunctionURLRequest, adapter AdapterFunc) (*events.LambdaFunctionURLStreamingResponse, error) {
	req, err := convertFunctionURLRequest(ctx, event)
	if err != nil {
		return nil, err
	}

	resCh := make(chan events.LambdaFunctionURLStreamingResponse)
	errCh := make(chan error)
	panicCh := make(chan any)

	go processRequestFunctionURLStreaming(ctx, req, adapter, resCh, errCh, panicCh)

	select {
	case res := <-resCh:
		return &res, nil
	case err = <-errCh:
		return nil, err
	case panicV := <-panicCh:
		panic(panicV)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func processRequestFunctionURLStreaming(ctx context.Context, req *http.Request, adapter AdapterFunc, resCh chan<- events.LambdaFunctionURLStreamingResponse, errCh chan<- error, panicCh chan<- any) {
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		if panicV := recover(); panicV != nil {
			panicCh <- panicV
		}

		close(panicCh)
		close(resCh)
		close(errCh)
		cancel()
	}()

	w := functionURLStreamingResponseWriter{
		headers:        make(http.Header),
		headersWritten: 0,
		resCh:          resCh,
	}

	defer w.Close()

	if err := adapter(ctx, req, &w); err != nil {
		errCh <- err
	}
}

func NewFunctionURLStreamingHandler(adapter AdapterFunc) func(context.Context, events.LambdaFunctionURLRequest) (*events.LambdaFunctionURLStreamingResponse, error) {
	return NewHandler(handleFunctionURLStreaming, adapter)
}

// endregion
