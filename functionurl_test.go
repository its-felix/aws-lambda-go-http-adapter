package aws_lambda_go_http_adapter

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/gofiber/fiber/v2"
	"github.com/its-felix/aws-lambda-go-http-adapter/adapter"
	"github.com/its-felix/aws-lambda-go-http-adapter/handler"
	"github.com/labstack/echo/v4"
	"io"
	"net/http"
	"reflect"
	"testing"
)

func newFunctionURLRequest() events.LambdaFunctionURLRequest {
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
				SourceIP:  "127.0.0.1",
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

func newFiberAdapter() handler.AdapterFunc {
	app := fiber.New()
	app.All("*", func(ctx *fiber.Ctx) error {
		result := make(map[string]string)
		result["Method"] = ctx.Method()
		result["URL"] = ctx.Request().URI().String()
		result["RemoteAddr"] = ctx.IP() + ":http" // fiber uses net.ResolveTCPAddr which resolves :http to :80
		result["Body"] = string(ctx.Body())

		return ctx.JSON(result)
	})

	return adapter.NewFiberAdapter(app)
}

func TestFunctionURLGET(t *testing.T) {
	adapters := map[string]handler.AdapterFunc{
		"vanilla": newVanillaAdapter(),
		"echo":    newEchoAdapter(),
		"fiber":   newFiberAdapter(),
	}

	for name, a := range adapters {
		t.Run(name, func(t *testing.T) {
			h := handler.NewFunctionURLHandler(a)

			req := newFunctionURLRequest()
			res, err := h(context.Background(), req)
			if err != nil {
				t.Error(err)
			}

			if res.StatusCode != http.StatusOK {
				t.Error("expected status to be 200")
			}

			if res.Headers["Content-Type"] != "application/json" {
				t.Error("expected Content-Type to be application/json")
			}

			if res.IsBase64Encoded {
				t.Error("expected body not to be base64 encoded")
			}

			body := make(map[string]string)
			_ = json.Unmarshal([]byte(res.Body), &body)

			expectedBody := map[string]string{
				"Method":     "POST",
				"URL":        "https://0dhg9709da0dhg9709da0dhg9709da.lambda-url.eu-central-1.on.aws/example?key=value",
				"RemoteAddr": "127.0.0.1:http",
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
