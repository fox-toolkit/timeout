[![Go Reference](https://pkg.go.dev/badge/github.com/fox-toolkit/timeout.svg)](https://pkg.go.dev/github.com/fox-toolkit/timeout)
[![tests](https://github.com/fox-toolkit/timeout/actions/workflows/tests.yaml/badge.svg)](https://github.com/fox-toolkit/timeout/actions?query=workflow%3Atests)
[![Go Report Card](https://goreportcard.com/badge/github.com/fox-toolkit/timeout)](https://goreportcard.com/report/github.com/fox-toolkit/timeout)
[![codecov](https://codecov.io/gh/fox-toolkit/timeout/branch/master/graph/badge.svg?token=D6qSTlzEcE)](https://codecov.io/gh/fox-toolkit/timeout)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/fox-toolkit/timeout)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/fox-toolkit/timeout)

# Timeout

> [!NOTE]
> This repository has been transferred from `github.com/tigerwill90/foxtimeout` to `github.com/fox-toolkit/timeout`.
> Existing users should update their imports and `go.mod` accordingly.

Timeout is a middleware for [Fox](https://github.com/fox-toolkit/fox) which ensure that a handler do not exceed the
configured timeout limit.

## Disclaimer
Timeout's API is closely tied to the Fox router, and it will only reach v1 when the router is stabilized.
During the pre-v1 phase, breaking changes may occur and will be documented in the release notes.

## Getting started
### Installation

````shell
go get -u github.com/fox-toolkit/timeout
````
### Feature
- Allows for custom timeout response to better suit specific use cases.
- Tightly integrates with the Fox ecosystem for enhanced performance and scalability.
- Supports dynamic timeout configuration on a per-route & per-request basis using custom `Resolver`.

### Usage
````go
package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/fox-toolkit/fox"
	"github.com/fox-toolkit/timeout"
)

func main() {
	f := fox.MustRouter(
		fox.DefaultOptions(),
		fox.WithMiddleware(
			timeout.Middleware(2*time.Second),
		),
	)

	f.MustAdd(fox.MethodGet, "/hello/{name}", func(c *fox.Context) {
		_ = c.String(http.StatusOK, fmt.Sprintf("Hello %s\n", c.Param("name")))
	})
	// Disable timeout the middleware for this route
	f.MustAdd(fox.MethodGet, "/download/{filepath}", DownloadHandler, timeout.OverrideHandler(timeout.NoTimeout))
	// Use 15s timeout instead of the global 2s for this route
	f.MustAdd(fox.MethodGet, "/workflow/{id}/start", WorkflowHandler, timeout.OverrideHandler(15*time.Second))

	if err := http.ListenAndServe(":8080", f); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalln(err)
	}
}
````
