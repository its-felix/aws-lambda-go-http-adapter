//go:build http_adapter_echo
// +build http_adapter_echo

package echo

import (
	"context"
	"github.com/its-felix/aws-lambda-go-http-adapter/handler"
	"github.com/labstack/echo/v4"
	"net/http"
)

type adapter struct {
	echo *echo.Echo
}

func (a adapter) adapterFunc(ctx context.Context, r *http.Request, w http.ResponseWriter) error {
	a.echo.ServeHTTP(w, r)
	return nil
}

func NewAdapter(delegate *echo.Echo) handler.AdapterFunc {
	return adapter{delegate}.adapterFunc
}
