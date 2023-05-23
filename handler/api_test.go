package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestHandler(handlerFunc http.HandlerFunc) func(context.Context, *http.Request) (*http.Response, error) {
	return NewHandler(
		func(ctx context.Context, event *http.Request, adapter AdapterFunc) (*http.Response, error) {
			w := httptest.NewRecorder()

			if err := adapter(ctx, event, w); err != nil {
				return nil, err
			}

			return w.Result(), nil
		},
		func(ctx context.Context, r *http.Request, w http.ResponseWriter) error {
			handlerFunc(w, r.WithContext(ctx))
			return nil
		},
	)
}

func TestGetSourceEvent(t *testing.T) {
	var caughtRequest *http.Request

	handler := newTestHandler(func(w http.ResponseWriter, r *http.Request) {
		event := GetSourceEvent(r.Context()).(*http.Request)
		caughtRequest = event
	})

	req := httptest.NewRequest(http.MethodGet, "/", strings.NewReader(""))
	res, err := handler(context.Background(), req)

	if caughtRequest != req {
		t.Error("GetSourceEvent returned the wrong value")
	}

	if res == nil {
		t.Error("response should not be nil")
	}

	if err != nil {
		t.Error("expected err to be nil")
	}
}

func TestWrapWithRecover(t *testing.T) {
	handler := newTestHandler(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic value")
	})

	handler = WrapWithRecover(handler, func(ctx context.Context, event *http.Request, panicValue any) (*http.Response, error) {
		return nil, errors.New(panicValue.(string))
	})

	req := httptest.NewRequest(http.MethodGet, "/", strings.NewReader(""))
	res, err := handler(context.Background(), req)

	if res != nil {
		t.Error("expected nil response")
	}

	if err.Error() != "test panic value" {
		t.Error("expected the handler to return an error 'test panic value'")
	}
}
