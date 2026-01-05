package common

import (
	"time"
)

// NowFunc returns the current time. Override in tests to inject fake clocks.
type NowFunc func() time.Time

// NowUTC is the default clock implementation.
var NowUTC NowFunc = func() time.Time {
	return time.Now().UTC()
}
