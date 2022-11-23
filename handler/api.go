package handler

import (
	"context"
	"net/http"
)

type httpContextKey string

var sourceEventContextKey httpContextKey = "github.com/its-felix/aws-lambda-go-adapter/httpadapter::sourceEventContextKey"

type RequestConverterFunc[In any] func(ctx context.Context, event In) (*http.Request, error)
type ResponseInitializerFunc[W http.ResponseWriter] func(ctx context.Context) W
type ResponseFinalizerFunc[W http.ResponseWriter, Out any] func(ctx context.Context, w W) (Out, error)
type AdapterFunc func(ctx context.Context, r *http.Request, w http.ResponseWriter) error

type RecoverFunc[In any, Out any] func(ctx context.Context, event In, panicValue any) (Out, error)

func NewHandler[In any, W http.ResponseWriter, Out any](reqConverter RequestConverterFunc[In], resInitializer ResponseInitializerFunc[W], resFinalizer ResponseFinalizerFunc[W, Out], adapter AdapterFunc) func(context.Context, In) (Out, error) {
	return func(ctx context.Context, event In) (Out, error) {
		ctx = WithSourceEvent(ctx, event)

		r, err := reqConverter(ctx, event)
		if err != nil {
			var def Out
			return def, err
		}

		w := resInitializer(ctx)
		if err = adapter(ctx, r, w); err != nil {
			var def Out
			return def, err
		}

		return resFinalizer(ctx, w)
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

func WithSourceEvent(ctx context.Context, event any) context.Context {
	return context.WithValue(ctx, sourceEventContextKey, event)
}

func GetSourceEvent(ctx context.Context) any {
	return ctx.Value(sourceEventContextKey)
}
