package aws_lambda_go_http_adapter

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/aws/aws-lambda-go/events"
	"github.com/gofiber/fiber/v2"
	"github.com/its-felix/aws-lambda-go-http-adapter/adapter"
	"github.com/its-felix/aws-lambda-go-http-adapter/handler"
	"github.com/labstack/echo/v4"
	"io"
	"net"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

func newFunctionURLRequest(sourceIp string) events.LambdaFunctionURLRequest {
	return events.LambdaFunctionURLRequest{
		Version:               "2.0",
		RawPath:               "/example",
		RawQueryString:        "key=value",
		Cookies:               []string{},
		Headers:               map[string]string{},
		QueryStringParameters: map[string]string{"key": "value"},
		RequestContext: events.LambdaFunctionURLRequestContext{
			AccountID:    "012345678912",
			RequestID:    "abcdefg",
			Authorizer:   nil,
			APIID:        "0dhg9709da0dhg9709da0dhg9709da",
			DomainName:   "0dhg9709da0dhg9709da0dhg9709da.lambda-url.eu-central-1.on.aws",
			DomainPrefix: "0dhg9709da0dhg9709da0dhg9709da",
			Time:         "",
			TimeEpoch:    0,
			HTTP: events.LambdaFunctionURLRequestContextHTTPDescription{
				Method:    "POST",
				Path:      "/example",
				Protocol:  "HTTP/1.1",
				SourceIP:  sourceIp,
				UserAgent: "Go-http-client/1.1",
			},
		},
		Body:            base64.StdEncoding.EncodeToString([]byte("hello world")),
		IsBase64Encoded: true,
	}
}

func newVanillaAdapter() handler.AdapterFunc {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		result := make(map[string]string)
		result["Method"] = r.Method
		result["URL"] = r.URL.String()
		result["RemoteAddr"] = r.RemoteAddr

		defer r.Body.Close()
		b, _ := io.ReadAll(r.Body)
		result["Body"] = string(b)

		enc := json.NewEncoder(w)
		_ = enc.Encode(result)
	})

	return adapter.NewVanillaAdapter(mux)
}

func newVanillaPanicAdapter() handler.AdapterFunc {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		panic("panic from test")
	})

	return adapter.NewVanillaAdapter(mux)
}

func newVanillaDelayedAdapter() handler.AdapterFunc {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)

		for i := 0; i < 10; i++ {
			_, _ = w.Write([]byte("pong"))
			time.Sleep(50 * time.Millisecond)
		}
	})

	return adapter.NewVanillaAdapter(mux)
}

func newEchoAdapter() handler.AdapterFunc {
	app := echo.New()
	app.Any("*", func(c echo.Context) error {
		r := c.Request()

		result := make(map[string]string)
		result["Method"] = r.Method
		result["URL"] = r.URL.String()
		result["RemoteAddr"] = r.RemoteAddr

		defer r.Body.Close()
		b, _ := io.ReadAll(r.Body)
		result["Body"] = string(b)

		c.Response().Header().Set("Content-Type", "application/json")

		return c.JSON(http.StatusOK, result)
	})

	return adapter.NewEchoAdapter(app)
}

func newEchoPanicAdapter() handler.AdapterFunc {
	app := echo.New()
	app.Any("*", func(c echo.Context) error {
		panic("panic from test")
	})

	return adapter.NewEchoAdapter(app)
}

func newEchoDelayedAdapter() handler.AdapterFunc {
	app := echo.New()
	app.Any("*", func(c echo.Context) error {
		w := c.Response()

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)

		for i := 0; i < 10; i++ {
			_, _ = w.Write([]byte("pong"))
			time.Sleep(50 * time.Millisecond)
		}

		return nil
	})

	return adapter.NewEchoAdapter(app)
}

func newFiberAdapter() handler.AdapterFunc {
	app := fiber.New()
	app.All("*", func(ctx *fiber.Ctx) error {
		result := make(map[string]string)
		result["Method"] = ctx.Method()
		result["URL"] = ctx.Request().URI().String()
		result["RemoteAddr"] = net.JoinHostPort(ctx.IP(), "http") // fiber uses net.ResolveTCPAddr which resolves :http to :80
		result["Body"] = string(ctx.Body())

		return ctx.JSON(result)
	})

	return adapter.NewFiberAdapter(app)
}

func newFiberPanicAdapter() handler.AdapterFunc {
	app := fiber.New()
	app.All("*", func(ctx *fiber.Ctx) error {
		panic("panic from test")
	})

	return adapter.NewFiberAdapter(app)
}

func newFiberDelayedAdapter() handler.AdapterFunc {
	app := fiber.New()
	app.All("*", func(ctx *fiber.Ctx) error {
		w := ctx.Response()

		w.Header.Set("Content-Type", "text/plain")
		w.Header.SetStatusCode(http.StatusOK)

		bw := w.BodyWriter()

		for i := 0; i < 10; i++ {
			_, _ = bw.Write([]byte("pong"))
			time.Sleep(50 * time.Millisecond)
		}

		return nil
	})

	return adapter.NewFiberAdapter(app)
}

type extractor[T any] interface {
	StatusCode(T) int
	Headers(T) map[string]string
	IsBase64Encoded(T) bool
	Body(T) string
}

type extractorNormal struct{}

func (extractorNormal) StatusCode(response events.LambdaFunctionURLResponse) int {
	return response.StatusCode
}

func (extractorNormal) Headers(response events.LambdaFunctionURLResponse) map[string]string {
	return response.Headers
}

func (extractorNormal) IsBase64Encoded(response events.LambdaFunctionURLResponse) bool {
	return response.IsBase64Encoded
}

func (extractorNormal) Body(response events.LambdaFunctionURLResponse) string {
	return response.Body
}

type extractorStreaming struct{}

func (extractorStreaming) StatusCode(response *events.LambdaFunctionURLStreamingResponse) int {
	return response.StatusCode
}

func (extractorStreaming) Headers(response *events.LambdaFunctionURLStreamingResponse) map[string]string {
	return response.Headers
}

func (extractorStreaming) IsBase64Encoded(*events.LambdaFunctionURLStreamingResponse) bool {
	return false
}

func (extractorStreaming) Body(response *events.LambdaFunctionURLStreamingResponse) string {
	defer func() {
		if rc, ok := response.Body.(io.Closer); ok {
			_ = rc.Close()
		}
	}()

	b, _ := io.ReadAll(response.Body)
	return string(b)
}

func TestFunctionURLPOST(t *testing.T) {
	adapters := map[string]handler.AdapterFunc{
		"vanilla": newVanillaAdapter(),
		"echo":    newEchoAdapter(),
		"fiber":   newFiberAdapter(),
	}

	for name, a := range adapters {
		t.Run(name, func(t *testing.T) {
			t.Run("normal", func(t *testing.T) {
				h := handler.NewFunctionURLHandler(a)
				runTestFunctionURLPOST[events.LambdaFunctionURLResponse](t, h, extractorNormal{})
			})

			t.Run("streaming", func(t *testing.T) {
				h := handler.NewFunctionURLStreamingHandler(a)
				runTestFunctionURLPOST[*events.LambdaFunctionURLStreamingResponse](t, h, extractorStreaming{})
			})
		})
	}
}

func runTestFunctionURLPOST[T any](t *testing.T, h func(context.Context, events.LambdaFunctionURLRequest) (T, error), ex extractor[T]) {
	sourceIps := map[string][2]string{
		"ipv4": {"127.0.0.1", "127.0.0.1:http"},
		"ipv6": {"::1", "[::1]:http"},
	}

	for name, inputPair := range sourceIps {
		t.Run(name, func(t *testing.T) {
			req := newFunctionURLRequest(inputPair[0])
			res, err := h(context.Background(), req)
			if err != nil {
				t.Error(err)
			}

			if ex.StatusCode(res) != http.StatusOK {
				t.Error("expected status to be 200")
			}

			if ex.Headers(res)["Content-Type"] != "application/json" {
				t.Error("expected Content-Type to be application/json")
			}

			if ex.IsBase64Encoded(res) {
				t.Error("expected body not to be base64 encoded")
			}

			body := make(map[string]string)
			_ = json.Unmarshal([]byte(ex.Body(res)), &body)

			expectedBody := map[string]string{
				"Method":     "POST",
				"URL":        "https://0dhg9709da0dhg9709da0dhg9709da.lambda-url.eu-central-1.on.aws/example?key=value",
				"RemoteAddr": inputPair[1],
				"Body":       "hello world",
			}

			if !reflect.DeepEqual(body, expectedBody) {
				t.Logf("expected: %v", expectedBody)
				t.Logf("actual: %v", body)
				t.Error("request/response didnt match")
			}
		})
	}
}

func TestFunctionURLWithPanicAndRecover(t *testing.T) {
	adapters := map[string]handler.AdapterFunc{
		"vanilla": newVanillaPanicAdapter(),
		"echo":    newEchoPanicAdapter(),
		"fiber":   newFiberPanicAdapter(),
	}

	for name, a := range adapters {
		t.Run(name, func(t *testing.T) {
			t.Run("normal", func(t *testing.T) {
				h := handler.NewFunctionURLHandler(a)
				h = handler.WrapWithRecover(h, func(ctx context.Context, event events.LambdaFunctionURLRequest, panicValue any) (events.LambdaFunctionURLResponse, error) {
					return events.LambdaFunctionURLResponse{}, errors.New(panicValue.(string))
				})

				runTestFunctionURLPanicAndRecover(t, h)
			})

			t.Run("streaming", func(t *testing.T) {
				h := handler.NewFunctionURLStreamingHandler(a)
				h = handler.WrapWithRecover(h, func(ctx context.Context, event events.LambdaFunctionURLRequest, panicValue any) (*events.LambdaFunctionURLStreamingResponse, error) {
					return nil, errors.New(panicValue.(string))
				})

				runTestFunctionURLPanicAndRecover(t, h)
			})
		})
	}
}

func runTestFunctionURLPanicAndRecover[T any](t *testing.T, h func(context.Context, events.LambdaFunctionURLRequest) (T, error)) {
	req := newFunctionURLRequest("127.0.0.1")
	_, err := h(context.Background(), req)
	if err == nil {
		t.Error("expected to receive an error")
	}

	if err.Error() != "panic from test" {
		t.Error("expected to receive error 'panic from test'")
	}
}

func TestFunctionURLDelayed(t *testing.T) {
	adapters := map[string]handler.AdapterFunc{
		"vanilla": newVanillaDelayedAdapter(),
		"echo":    newEchoDelayedAdapter(),
		"fiber":   newFiberDelayedAdapter(),
	}

	for name, a := range adapters {
		t.Run(name, func(t *testing.T) {
			t.Run("normal", func(t *testing.T) {
				h := handler.NewFunctionURLHandler(a)
				runTestFunctionURLDelayed[events.LambdaFunctionURLResponse](t, h, extractorNormal{})
			})

			t.Run("streaming", func(t *testing.T) {
				h := handler.NewFunctionURLStreamingHandler(a)
				runTestFunctionURLDelayed[*events.LambdaFunctionURLStreamingResponse](t, h, extractorStreaming{})
			})
		})
	}
}

func runTestFunctionURLDelayed[T any](t *testing.T, h func(context.Context, events.LambdaFunctionURLRequest) (T, error), ex extractor[T]) {
	req := newFunctionURLRequest("127.0.0.1")
	res, err := h(context.Background(), req)
	if err != nil {
		t.Error(err)
	}

	if ex.StatusCode(res) != http.StatusOK {
		t.Error("expected status to be 200")
	}

	if ex.Headers(res)["Content-Type"] != "text/plain" {
		t.Error("expected Content-Type to be text/plain")
	}

	if ex.IsBase64Encoded(res) {
		t.Error("expected body not to be base64 encoded")
	}

	body := ex.Body(res)
	expectedBody := strings.Repeat("pong", 10)

	if body != expectedBody {
		t.Logf("expected: %v", expectedBody)
		t.Logf("actual: %v", body)
		t.Error("request/response didnt match")
	}
}
