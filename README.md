# aws-lambda-go-http-adapter
Simple HTTP adapter for AWS Lambda

[![Go Reference](https://pkg.go.dev/badge/github.com/its-felix/aws-lambda-go-http-adapter.svg)](https://pkg.go.dev/github.com/its-felix/aws-lambda-go-http-adapter)
[![Go Report](https://goreportcard.com/badge/github.com/its-felix/aws-lambda-go-http-adapter?style=flat-square)](https://goreportcard.com/report/github.com/its-felix/aws-lambda-go-http-adapter)

## Builtin support for these event formats:
- AWS Lambda Function URL (both normal and streaming)
- API Gateway (v1)
- API Gateway (v2)

## Builtin support for these HTTP frameworks:
- `net/http`
- [Echo](https://github.com/labstack/echo)
- [Fiber](https://github.com/gofiber/fiber)

## Usage
### Creating the Adapter
#### net/http
```golang
package main

import (
	"github.com/its-felix/aws-lambda-go-http-adapter/adapter/vanilla"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pong"))
	})
	
	adapter := vanilla.NewAdapter(mux)
}
```

#### Echo
```golang
package main

import (
	"github.com/labstack/echo/v4"
	echoadapter "github.com/its-felix/aws-lambda-go-http-adapter/adapter/echo"
	"net/http"
)

func main() {
	e := echo.New()
	e.Add("GET", "/ping", func(c echo.Context) error {
		return c.String(200, "pong")
	})
	
	adapter := echoadapter.NewAdapter(e)
}
```

#### Fiber
```golang
package main

import (
	"github.com/gofiber/fiber/v2"
	fiberadapter "github.com/its-felix/aws-lambda-go-http-adapter/adapter/fiber"
	"net/http"
)

func main() {
	app := fiber.New()
	app.Get("/ping", func(ctx *fiber.Ctx) error {
		return ctx.SendString("pong")
	})

	adapter := fiberadapter.NewAdapter(app)
}
```

### Creating the Handler
#### API Gateway V1
```golang
package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/its-felix/aws-lambda-go-http-adapter/handler"
)

func main() {
	adapter := [...] // see above
	h := handler.NewAPIGatewayV1Handler(adapter)
	
	lambda.Start(h)
}
```

#### API Gateway V2
```golang
package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/its-felix/aws-lambda-go-http-adapter/handler"
)

func main() {
	adapter := [...] // see above
	h := handler.NewAPIGatewayV2Handler(adapter)
	
	lambda.Start(h)
}
```

#### Lambda Function URL (normal)
```golang
package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/its-felix/aws-lambda-go-http-adapter/handler"
)

func main() {
	adapter := [...] // see above
	h := handler.NewFunctionURLHandler(adapter)
	
	lambda.Start(h)
}
```

#### Lambda Function URL (streaming)
```golang
package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/its-felix/aws-lambda-go-http-adapter/handler"
)

func main() {
	adapter := [...] // see above
	h := handler.NewFunctionURLStreamingHandler(adapter)
	
	lambda.Start(h)
}
```

### Accessing the source event
#### Fiber
```golang
package main

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/gofiber/fiber/v2"
	fiberadapter "github.com/its-felix/aws-lambda-go-http-adapter/adapter/fiber"
)

func main() {
	app := fiber.New()
	app.Get("/ping", func(ctx *fiber.Ctx) error {
		event := fiberadapter.GetSourceEvent(ctx)
		switch event := event.(type) {
		case events.APIGatewayProxyRequest:
			// do something
		case events.APIGatewayV2HTTPRequest:
			// do something
		case events.LambdaFunctionURLRequest:
			// do something
		}
		
		return ctx.SendString("pong")
	})
}
```

#### Others
```golang
package main

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/its-felix/aws-lambda-go-http-adapter/handler"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		event := handler.GetSourceEvent(r.Context())
		switch event := event.(type) {
		case events.APIGatewayProxyRequest:
			// do something
		case events.APIGatewayV2HTTPRequest:
			// do something
		case events.LambdaFunctionURLRequest:
			// do something
		}
		
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pong"))
	})
}
```

### Handle panics
To handle panics, first create the handler as described above. You can then wrap the handler to handle panics like so:
```golang
package main

import (
	"context"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/events"
	"github.com/its-felix/aws-lambda-go-http-adapter/handler"
)

func main() {
	adapter := [...] // see above
	h := [...] // see above
	h = handler.WrapWithRecover(h, func(ctx context.Context, event events.APIGatewayV2HTTPRequest, panicValue any) (events.APIGatewayV2HTTPResponse, error) {
		return events.APIGatewayV2HTTPResponse{
			StatusCode:        500,
			Headers:           make(map[string]string),
			Body:              fmt.Sprintf("Unexpected error: %v", panicValue),
		}, nil
	})
	
	lambda.Start(h)
}
```

## Extending for other lambda event formats:
Have a look at the existing event handlers:
- [API Gateway V1](./handler/apigwv1.go)
- [API Gateway V2](./handler/apigwv2.go)
- [Lambda Function URL](./handler/functionurl.go)

## Extending for other frameworks
Have a look at the existing adapters:
- [net/http](./adapter/vanilla/vanilla.go)
- [Echo](./adapter/echo/echo.go)
- [Fiber](./adapter/fiber/fiber.go)

## Build Tags
You can opt-in to enable partial build by using the build-tag `lambdahttpadapter.partial`.

Once this build-tag is present, the following build-tags are available:
- `lambdahttpadapter.vanilla` (enables the vanilla adapter)
- `lambdahttpadapter.echo` (enables the echo adapter)
- `lambdahttpadapter.fiber` (enables the fiber adapter)
- `lambdahttpadapter.apigwv1` (enables API Gateway V1 handler)
- `lambdahttpadapter.apigwv2` (enables API Gateway V2 handler)
- `lambdahttpadapter.functionurl` (enables Lambda Function URL handler)

Also note that Lambda Function URL in Streaming-Mode requires the following build-tag to be set:
- `lambda.norpc`