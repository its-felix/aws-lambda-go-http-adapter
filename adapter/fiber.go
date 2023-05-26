//go:build !lambdahttpadapter.partial || (lambdahttpadapter.partial && lambdahttpadapter.fiber)

package adapter

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/its-felix/aws-lambda-go-http-adapter/handler"
	"github.com/valyala/fasthttp"
	"io"
	"net"
	"net/http"
	"strings"
)

const contextUserValueKey = "github.com/its-felix/aws-lambda-go-http-adapter/adapter/fiber::contextUserValueKey"

type fiberAdapter struct {
	app     *fiber.App
	handler fasthttp.RequestHandler
}

func (a fiberAdapter) adapterFunc(ctx context.Context, r *http.Request, w http.ResponseWriter) error {
	httpReq := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(httpReq)

	// protocol, method, uri, host
	httpReq.Header.SetProtocol(r.Proto)
	httpReq.Header.SetMethod(r.Method)
	httpReq.SetRequestURI(r.URL.Scheme + "://" + r.RequestURI)
	httpReq.SetHost(r.Host)

	// body
	if r.Body != nil {
		defer r.Body.Close()
		written, err := io.Copy(httpReq.BodyWriter(), r.Body)
		if err != nil {
			return err
		}

		httpReq.Header.SetContentLength(int(written))
	}

	// headers
	for k, values := range r.Header {
		for _, v := range values {
			switch k {
			case fiber.HeaderHost,
				fiber.HeaderContentType,
				fiber.HeaderUserAgent,
				fiber.HeaderContentLength,
				fiber.HeaderConnection:
				httpReq.Header.Set(k, v)
			default:
				httpReq.Header.Add(k, v)
			}
		}
	}

	// remoteAddr
	remoteAddr, err := net.ResolveTCPAddr("tcp", r.RemoteAddr)
	if err != nil {
		return err
	}

	var fctx fasthttp.RequestCtx
	fctx.Init(httpReq, remoteAddr, nil)
	defer fasthttp.ReleaseResponse(&fctx.Response)

	fctx.SetUserValue(contextUserValueKey, ctx)

	a.handler(&fctx)

	fctx.Response.Header.VisitAll(func(key, value []byte) {
		k := utils.UnsafeString(key)

		for _, v := range strings.Split(utils.UnsafeString(value), ",") {
			w.Header().Add(k, v)
		}
	})

	w.WriteHeader(fctx.Response.StatusCode())
	// release handled in defer
	err = fctx.Response.BodyWriteTo(w)

	return err
}

func NewFiberAdapter(delegate *fiber.App) handler.AdapterFunc {
	return fiberAdapter{delegate, delegate.Handler()}.adapterFunc
}

func GetContextFiber(ctx *fiber.Ctx) context.Context {
	return ctx.Context().Value(contextUserValueKey).(context.Context)
}

func GetSourceEventFiber(ctx *fiber.Ctx) any {
	return handler.GetSourceEvent(GetContextFiber(ctx))
}
