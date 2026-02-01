// Copyright 2023 Sylvain Müller. All rights reserved.
// Mount of this source code is governed by a MIT license that can be found
// at https://github.com/fox-toolkit/timeout/blob/master/LICENSE.txt.
//
// This package is based on the Go standard library, see the LICENSE file
// at https://github.com/golang/go/blob/master/LICENSE.

package timeout

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fox-toolkit/fox"
)

var (
	bufp = sync.Pool{
		New: func() any {
			return bytes.NewBuffer(nil)
		},
	}
)

// Timeout is a middleware that ensure HTTP handlers don't exceed the configured timeout duration.
type Timeout struct {
	cfg *config
	dt  time.Duration
}

// Middleware returns a [fox.MiddlewareFunc] that runs handlers with the given time limit.
//
// The middleware calls the next handler to handle each request, but if a call runs for longer than its time limit,
// the handler responds with a 503 Service Unavailable error and the given message in its body (if a custom response
// handler is not configured). After such a timeout, writes by the handler to its ResponseWriter will return [http.ErrHandlerTimeout].
//
// The timeout middleware supports the [http.Pusher] interface but does not support the [http.Hijacker] or [http.Flusher] interfaces.
//
// Individual routes can override the timeout duration using the [OverrideHandler] option. It's also possible to set the read
// and write deadline for individual route using the [OverrideRead] and [OverrideWrite] option.
// If dt <= 0 (or NoTimeout), this is a passthrough middleware but per-route options remain effective.
func Middleware(dt time.Duration, opts ...Option) fox.MiddlewareFunc {
	return create(dt, opts...).run
}

func create(dt time.Duration, opts ...Option) *Timeout {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt.apply(cfg)
	}

	return &Timeout{
		dt:  dt,
		cfg: cfg,
	}
}

// run is the internal handler that applies the timeout logic.
func (t *Timeout) run(next fox.HandlerFunc) fox.HandlerFunc {
	return func(c *fox.Context) {
		t.setDeadline(c)
		dt := t.resolveTimeout(c)
		if dt <= 0 {
			next(c)
			return
		}

		ctx, cancel := context.WithTimeout(c.Request().Context(), dt)
		defer cancel()

		req := c.Request().WithContext(ctx)
		done := make(chan struct{})
		panicChan := make(chan any, 1)

		w := c.Writer()
		buf := bufp.Get().(*bytes.Buffer)
		defer bufp.Put(buf)
		buf.Reset()

		tw := &timeoutWriter{
			w:       w,
			headers: make(http.Header),
			req:     req,
			code:    http.StatusOK,
			buf:     buf,
		}

		cp := c.CloneWith(tw, req)

		go func() {
			defer func() {
				cp.Close()
				if p := recover(); p != nil {
					panicChan <- p
				}
			}()
			next(cp)
			close(done)
		}()

		select {
		case p := <-panicChan:
			panic(p)
		case <-done:
			tw.mu.Lock()
			defer tw.mu.Unlock()
			dst := w.Header()
			maps.Copy(dst, tw.headers)
			w.WriteHeader(tw.code)
			_, _ = w.Write(tw.buf.Bytes())
		case <-ctx.Done():
			tw.mu.Lock()
			defer tw.mu.Unlock()
			switch err := ctx.Err(); err {
			case context.DeadlineExceeded:
				tw.err = http.ErrHandlerTimeout
			default:
				tw.err = err
			}
			t.cfg.resp(c)
		}
	}
}

func (t *Timeout) resolveTimeout(c *fox.Context) time.Duration {
	if dt, ok := unwrapRouteTimeout(c.Route(), hKey{}); ok {
		return dt
	}
	return t.dt
}

func (t *Timeout) setDeadline(c *fox.Context) {
	// Errors are intentionally ignored: the underlying connection may not support deadlines
	// (e.g., http.ErrNotSupported), and there's no actionable recovery in this context.
	if dt, ok := unwrapRouteTimeout(c.Route(), rKey{}); ok {
		_ = c.Writer().SetReadDeadline(time.Now().Add(dt))
	}
	if dt, ok := unwrapRouteTimeout(c.Route(), wKey{}); ok {
		_ = c.Writer().SetWriteDeadline(time.Now().Add(dt))
	}
}

func checkWriteHeaderCode(code int) {
	if code < 100 || code > 999 {
		panic(fmt.Sprintf("invalid status code %d", code))
	}
}

func relevantCaller() runtime.Frame {
	pc := make([]uintptr, 16)
	n := runtime.Callers(1, pc)
	frames := runtime.CallersFrames(pc[:n])
	var frame runtime.Frame
	for {
		f, more := frames.Next()
		if !strings.HasPrefix(f.Function, "github.com/fox-toolkit/timeout.") {
			return f
		}
		if !more {
			break
		}
	}
	return frame
}
