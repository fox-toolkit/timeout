// Copyright 2023 Sylvain Müller. All rights reserved.
// Mount of this source code is governed by a MIT license that can be found
// at https://github.com/fox-toolkit/timeout/blob/master/LICENSE.txt.

package timeout

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fox-toolkit/fox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func success201response(c *fox.Context) {
	time.Sleep(10 * time.Millisecond)
	_ = c.String(http.StatusCreated, fmt.Sprintf("%s\n", http.StatusText(http.StatusCreated)))
}

func TestMiddleware_WithTimeout(t *testing.T) {
	f, err := fox.NewRouter(fox.WithMiddleware(Middleware(50 * time.Microsecond)))
	require.NoError(t, err)
	f.MustAdd(fox.MethodGet, "/foo", success201response)

	req := httptest.NewRequest(http.MethodGet, "/foo", nil)
	w := httptest.NewRecorder()
	f.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, fmt.Sprintf("%s\n", http.StatusText(http.StatusServiceUnavailable)), w.Body.String())
}

func TestMiddleware_WithoutTimeout(t *testing.T) {
	f, err := fox.NewRouter(fox.WithMiddleware(Middleware(1 * time.Second)))
	require.NoError(t, err)
	f.MustAdd(fox.MethodGet, "/foo", success201response)

	req := httptest.NewRequest(http.MethodGet, "/foo", nil)
	w := httptest.NewRecorder()
	f.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, fmt.Sprintf("%s\n", http.StatusText(http.StatusCreated)), w.Body.String())
}

func timeoutResponse(c *fox.Context) {
	http.Error(c.Writer(), http.StatusText(http.StatusRequestTimeout), http.StatusRequestTimeout)
}

func TestMiddleware_WithResponse(t *testing.T) {
	f, err := fox.NewRouter(fox.WithMiddleware(Middleware(50*time.Microsecond, WithResponse(timeoutResponse))))
	require.NoError(t, err)
	f.MustAdd(fox.MethodGet, "/foo", success201response)

	req := httptest.NewRequest(http.MethodGet, "/foo", nil)
	w := httptest.NewRecorder()
	f.ServeHTTP(w, req)

	assert.Equal(t, http.StatusRequestTimeout, w.Code)
	assert.Equal(t, fmt.Sprintf("%s\n", http.StatusText(http.StatusRequestTimeout)), w.Body.String())
}

func panicResponse(c *fox.Context) {
	panic("test")
}

func TestMiddleware_WithPanic(t *testing.T) {
	f, err := fox.NewRouter(
		fox.WithMiddleware(
			fox.RecoveryWithFunc(slog.DiscardHandler, func(c *fox.Context, err any) {
				if !c.Writer().Written() {
					http.Error(c.Writer(), http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}),
			Middleware(1*time.Second, WithResponse(timeoutResponse)),
		),
	)
	require.NoError(t, err)
	f.MustAdd(fox.MethodGet, "/foo", panicResponse)

	req := httptest.NewRequest(http.MethodGet, "/foo", nil)
	w := httptest.NewRecorder()
	f.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, fmt.Sprintf("%s\n", http.StatusText(http.StatusInternalServerError)), w.Body.String())
}

func TestMiddleware_NoTimeout(t *testing.T) {
	f, err := fox.NewRouter(fox.WithMiddleware(Middleware(0)))
	require.NoError(t, err)
	f.MustAdd(fox.MethodGet, "/foo", success201response)

	req := httptest.NewRequest(http.MethodGet, "/foo", nil)
	w := httptest.NewRecorder()
	f.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, fmt.Sprintf("%s\n", http.StatusText(http.StatusCreated)), w.Body.String())
}

func TestMiddleware_ErrNotSupported(t *testing.T) {
	f, err := fox.NewRouter(fox.WithMiddleware(Middleware(1 * time.Second)))
	require.NoError(t, err)
	f.MustAdd(fox.MethodGet, "/foo", func(c *fox.Context) {
		assert.ErrorIs(t, c.Writer().FlushError(), http.ErrNotSupported)
		_, _, hijErr := c.Writer().Hijack()
		assert.ErrorIs(t, hijErr, http.ErrNotSupported)
		assert.ErrorIs(t, c.Writer().SetReadDeadline(time.Now()), http.ErrNotSupported)
		assert.ErrorIs(t, c.Writer().SetWriteDeadline(time.Now()), http.ErrNotSupported)
	})

	req := httptest.NewRequest(http.MethodGet, "/foo", nil)
	w := httptest.NewRecorder()
	f.ServeHTTP(w, req)
}

func TestMiddleware_WithHandlerTimeout(t *testing.T) {
	f, err := fox.NewRouter(fox.WithMiddleware(Middleware(1 * time.Millisecond)))
	require.NoError(t, err)
	f.MustAdd(fox.MethodGet, "/foo", success201response, OverrideHandler(2*time.Second))

	req := httptest.NewRequest(http.MethodGet, "/foo", nil)
	w := httptest.NewRecorder()
	f.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, fmt.Sprintf("%s\n", http.StatusText(http.StatusCreated)), w.Body.String())
}

func TestMiddleware_WithDisableTimeout(t *testing.T) {
	f, err := fox.NewRouter(fox.WithMiddleware(Middleware(1 * time.Millisecond)))
	require.NoError(t, err)
	f.MustAdd(fox.MethodGet, "/foo", success201response, OverrideHandler(NoTimeout))

	req := httptest.NewRequest(http.MethodGet, "/foo", nil)
	w := httptest.NewRecorder()
	f.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, fmt.Sprintf("%s\n", http.StatusText(http.StatusCreated)), w.Body.String())
}

func TestMiddleware_WithReadTimeout(t *testing.T) {
	f, err := fox.NewRouter(fox.WithMiddleware(Middleware(NoTimeout)))
	require.NoError(t, err)

	called := false
	f.MustAdd(fox.MethodPost, "/foo", func(c *fox.Context) {
		buf := make([]byte, 1024)
		_, err := c.Request().Body.Read(buf)
		if err != nil {
			called = true
			assert.Contains(t, err.Error(), "i/o timeout")
			http.Error(c.Writer(), err.Error(), http.StatusRequestTimeout)
			return
		}
		c.Writer().WriteHeader(http.StatusOK)
	}, OverrideRead(50*time.Millisecond))

	srv := httptest.NewServer(f)
	defer srv.Close()

	pr, pw := io.Pipe()
	go func() {
		// Slow writer: sends data too slowly, causing read timeout on server
		time.Sleep(200 * time.Millisecond)
		_, _ = pw.Write([]byte("hello"))
		pw.Close()
	}()

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/foo", pr)
	require.NoError(t, err)

	_, _ = http.DefaultClient.Do(req)
	assert.True(t, called)
}

func TestMiddleware_WithWriteTimeout(t *testing.T) {
	f, err := fox.NewRouter(fox.WithMiddleware(Middleware(NoTimeout)))
	require.NoError(t, err)

	f.MustAdd(fox.MethodGet, "/foo", func(c *fox.Context) {
		data := bytes.Repeat([]byte("x"), 10*1024*1024)
		_, _ = c.Writer().Write(data)
	}, OverrideWrite(50*time.Millisecond))

	srv := httptest.NewServer(f)
	defer srv.Close()

	addr := srv.Listener.Addr().String()
	conn, err := net.Dial("tcp", addr)
	require.NoError(t, err)
	defer conn.Close()

	_, err = fmt.Fprintf(conn, "GET /foo HTTP/1.1\r\nHost: %s\r\n\r\n", addr)
	require.NoError(t, err)

	time.Sleep(1 * time.Second) // let TCP buffer fill

	_ = conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	n, _ := io.Copy(io.Discard, conn)
	assert.Less(t, n, int64(10*1024*1024))
}

func ExampleOverrideHandler() {
	f, err := fox.NewRouter(
		fox.WithMiddleware(Middleware(2 * time.Second)),
	)
	if err != nil {
		panic(err)
	}

	f.MustAdd(fox.MethodGet, "/hello/{name}", func(c *fox.Context) {
		_ = c.String(http.StatusOK, fmt.Sprintf("hello %s\n", c.Param("name")))
	})

	f.MustAdd(fox.MethodGet, "/long", func(c *fox.Context) {
		time.Sleep(10 * time.Second)
		c.Writer().WriteHeader(http.StatusOK)
	}, OverrideHandler(12*time.Second))

	f.MustAdd(fox.MethodGet, "/no-timeout", func(c *fox.Context) {
		c.Writer().WriteHeader(http.StatusOK)
	}, OverrideHandler(NoTimeout))
}
