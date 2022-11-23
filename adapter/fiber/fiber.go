package fiber

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

const sourceEventUserValueKey = "github.com/its-felix/aws-lambda-go-adapter-fiber::sourceEvent"

type adapter struct {
	app *fiber.App
}

func (a adapter) adapterFunc(ctx context.Context, r *http.Request, w http.ResponseWriter) error {
	httpReq := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(httpReq)

	// protocol, method, uri, host
	httpReq.Header.SetProtocol(r.Proto)
	httpReq.Header.SetMethod(r.Method)
	httpReq.SetRequestURI(r.RequestURI)
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
	fctx.SetUserValue(sourceEventUserValueKey, handler.GetSourceEvent(ctx))

	a.app.Handler()(&fctx)

	fctx.Response.Header.VisitAll(func(key, value []byte) {
		k := utils.UnsafeString(key)

		for _, v := range strings.Split(utils.UnsafeString(value), ",") {
			w.Header().Add(k, v)
		}
	})

	w.WriteHeader(fctx.Response.StatusCode())
	_, _ = w.Write(fctx.Response.Body())

	return nil
}

func NewAdapter(delegate *fiber.App) handler.AdapterFunc {
	return adapter{delegate}.adapterFunc
}

func GetSourceEvent(ctx *fiber.Ctx) any {
	return ctx.Context().UserValue(sourceEventUserValueKey)
}
