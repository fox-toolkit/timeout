package foxtimeout

import (
	"time"

	"github.com/tigerwill90/fox"
)

type key struct{}

var (
	timeoutKey      key
	readTimeoutKey  key
	writeTimeoutKey key
)

const NoTimeout = time.Duration(0)

// HandlerTimeout returns a RouteOption that sets a custom timeout duration for a specific route.
// This allows individual routes to have different timeout values than the global timeout.
// Passing a value <= 0 (or NoTimeout) disables the timeout for this route.
func HandlerTimeout(dt time.Duration) fox.RouteOption {
	return fox.WithAnnotation(timeoutKey, dt)
}

// ReadTimeout returns a RouteOption that sets the read deadline for the underlying connection.
// This controls how long the server will wait for the client to send request data.
// A zero duration is not allowed and will return an error during route registration.
func ReadTimeout(dt time.Duration) fox.RouteOption {
	return fox.WithAnnotation(readTimeoutKey, dt)
}

// WriteTimeout returns a RouteOption that sets the write deadline for the underlying connection.
// This controls how long the server will wait before timing out writes to the client.
// A zero duration is not allowed and will return an error during route registration.
func WriteTimeout(dt time.Duration) fox.RouteOption {
	return fox.WithAnnotation(writeTimeoutKey, dt)
}

func unwrapRouteTimeout(r *fox.Route, k key) (time.Duration, bool) {
	if r != nil {
		dt := r.Annotation(k)
		if dt != nil {
			return dt.(time.Duration), true
		}
	}
	return 0, false
}
