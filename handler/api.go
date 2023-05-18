package handler

import (
	"context"
	"net/http"
)

type httpContextKey string

var sourceEventContextKey httpContextKey = "github.com/its-felix/aws-lambda-go-http-adapter/api::sourceEventContextKey"

type AdapterFunc func(ctx context.Context, r *http.Request, w http.ResponseWriter) error
type HandlerFunc[In any, Out any] func(ctx context.Context, event In, adapter AdapterFunc) (Out, error)
type RecoverFunc[In any, Out any] func(ctx context.Context, event In, panicValue any) (Out, error)

func NewHandler[In any, Out any](handlerFunc HandlerFunc[In, Out], adapter AdapterFunc) func(context.Context, In) (Out, error) {
	return func(ctx context.Context, event In) (Out, error) {
		ctx = context.WithValue(ctx, sourceEventContextKey, event)
		return handlerFunc(ctx, event, adapter)
	}
}

func WrapWithRecover[In any, Out any](handler func(context.Context, In) (Out, error), recoverFunc RecoverFunc[In, Out]) func(context.Context, In) (Out, error) {
	return func(ctx context.Context, event In) (Out, error) {
		var out Out
		var err error

		func() {
			defer func() {
				if panicV := recover(); panicV != nil {
					out, err = recoverFunc(ctx, event, panicV)
				}
			}()

			out, err = handler(ctx, event)
		}()

		return out, err
	}
}

func GetSourceEvent(ctx context.Context) any {
	return ctx.Value(sourceEventContextKey)
}
