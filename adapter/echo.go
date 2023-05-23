//go:build !lambdahttpadapter.partial || (lambdahttpadapter.partial && lambdahttpadapter.echo)

package adapter

import (
	"context"
	"github.com/its-felix/aws-lambda-go-http-adapter/handler"
	"github.com/labstack/echo/v4"
	"net/http"
)

type echoAdapter struct {
	echo *echo.Echo
}

func (a echoAdapter) adapterFunc(ctx context.Context, r *http.Request, w http.ResponseWriter) error {
	a.echo.ServeHTTP(w, r)
	return nil
}

func NewEchoAdapter(delegate *echo.Echo) handler.AdapterFunc {
	return echoAdapter{delegate}.adapterFunc
}
