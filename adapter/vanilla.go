//go:build !lambdahttpadapter.partial || (lambdahttpadapter.partial && lambdahttpadapter.vanilla)

package adapter

import (
	"context"
	"github.com/its-felix/aws-lambda-go-http-adapter/handler"
	"net/http"
)

type vanillaAdapter struct {
	http.Handler
}

func (a vanillaAdapter) adapterFunc(ctx context.Context, r *http.Request, w http.ResponseWriter) error {
	a.ServeHTTP(w, r)
	return nil
}

func NewVanillaAdapter(delegate http.Handler) handler.AdapterFunc {
	return vanillaAdapter{delegate}.adapterFunc
}
