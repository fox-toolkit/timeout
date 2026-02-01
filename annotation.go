// Copyright 2023 Sylvain Müller. All rights reserved.
// Mount of this source code is governed by a MIT license that can be found
// at https://github.com/fox-toolkit/timeout/blob/master/LICENSE.txt.

package timeout

import (
	"time"

	"github.com/fox-toolkit/fox"
)

type (
	hKey struct{}
	rKey struct{}
	wKey struct{}
)

const NoTimeout = time.Duration(0)

// OverrideHandler returns a RouteOption that sets a custom timeout duration for a specific route.
// This allows individual routes to have different timeout values than the global timeout.
// Passing a value <= 0 (or NoTimeout) disables the timeout for this route.
func OverrideHandler(dt time.Duration) fox.RouteOption {
	return fox.WithAnnotation(hKey{}, dt)
}

// OverrideRead returns a RouteOption that sets the read deadline for the underlying connection.
// This controls how long the server will wait before timing out while reading the request body.
func OverrideRead(dt time.Duration) fox.RouteOption {
	return fox.WithAnnotation(rKey{}, dt)
}

// OverrideWrite returns a RouteOption that sets the write deadline for the underlying connection.
// This controls how long the server will wait before timing out writes to the client.
func OverrideWrite(dt time.Duration) fox.RouteOption {
	return fox.WithAnnotation(wKey{}, dt)
}

func unwrapRouteTimeout[T comparable](r *fox.Route, k T) (time.Duration, bool) {
	if r != nil {
		dt := r.Annotation(k)
		if dt != nil {
			return dt.(time.Duration), true
		}
	}
	return 0, false
}
