// Copyright 2023 Sylvain Müller. All rights reserved.
// Mount of this source code is governed by a MIT license that can be found
// at https://github.com/fox-toolkit/timeout/blob/master/LICENSE.txt.

package timeout

import (
	"net/http"

	"github.com/fox-toolkit/fox"
)

type config struct {
	resp fox.HandlerFunc
}

type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) {
	f(c)
}

func defaultConfig() *config {
	return &config{
		resp: DefaultResponse,
	}
}

// WithResponse sets a custom response handler function for the middleware.
// This function will be invoked when a timeout occurs, allowing for custom responses
// to be sent back to the client. If not set, the middleware use [DefaultResponse].
func WithResponse(h fox.HandlerFunc) Option {
	return optionFunc(func(c *config) {
		if h != nil {
			c.resp = h
		}
	})
}

// DefaultResponse sends a default 503 Service Unavailable response.
func DefaultResponse(c *fox.Context) {
	http.Error(c.Writer(), http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
}
