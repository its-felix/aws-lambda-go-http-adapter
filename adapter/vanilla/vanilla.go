package vanilla

import (
	"context"
	"github.com/its-felix/aws-lambda-go-http-adapter/handler"
	"net/http"
)

type adapter struct {
	http.Handler
}

func (a adapter) adapterFunc(ctx context.Context, r *http.Request, w http.ResponseWriter) error {
	a.ServeHTTP(w, r)
	return nil
}

func NewAdapter(delegate http.Handler) handler.AdapterFunc {
	return adapter{delegate}.adapterFunc
}
