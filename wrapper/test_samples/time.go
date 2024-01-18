package testsamples

import (
	time "github.com/itzloop/promwrapgen/wrapper/test_samples/mytime"
	time2 "time"
)

type TimeConflict interface {
	Method1(t1 time.Time, t2 time.Time) time2.Time
}
