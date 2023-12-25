// This code is generate by github.com/itzloop/promwrapgen. DO NOT EDIT!
package main

import (
	"net"
	"time"
)

// IPUtilPrometheusWrapperImpl wraps IPUtil and adds metrics like:
// 1. success count
// 2. error count
// 3. total count
// 4. duration
type IPUtilPrometheusWrapperImpl struct {
	// TODO what are fields are required
	wrapped IPUtil
	metrics interface {
		// Failure will be called when err != nil passing the duration and err to it
		Failure(pkg, intr, method string, duration time.Duration, err error)

		// Success will be called if err == nil passing the duration to it
		Success(pkg, intr, method string, duration time.Duration)

		// Total will be called as soon as the function is called.
		Total(pkg, intr, method string)
	}
}

func NewIPUtilPrometheusWrapperImpl(
	wrapped IPUtil,
	metrics interface {
		Failure(pkg, intr, method string, duration time.Duration, err error)
		Success(pkg, intr, method string, duration time.Duration)
		Total(pkg, intr, method string)
	},
) *IPUtilPrometheusWrapperImpl {
	return &IPUtilPrometheusWrapperImpl{
		wrapped: wrapped,
		metrics: metrics,
	}
}

// PublicIP wraps another instance of IPUtil and
// adds prometheus metrics. See PublicIP on IPUtilPrometheusWrapperImpl.wrapped for
// more information.
func (w *IPUtilPrometheusWrapperImpl) PublicIP() (net.IP, error) {
	// TODO time package conflicts
	start_49a52cf7 := time.Now()
	w.metrics.Total("main", "IPUtil", "PublicIP")
	a, err := w.wrapped.PublicIP()
	duration_58a84e00 := time.Since(start_49a52cf7)
	if err != nil {
		w.metrics.Failure("main", "IPUtil", "PublicIP", duration_58a84e00, err)
		// TODO find a way to add default values here and return the error. for now return the same thing :)
		return a, err
	}

	// TODO if method has no error does success matter or not?
	w.metrics.Success("main", "IPUtil", "PublicIP", duration_58a84e00)

	return a, err
}

// LocalIPs wraps another instance of IPUtil and
// adds prometheus metrics. See LocalIPs on IPUtilPrometheusWrapperImpl.wrapped for
// more information.
func (w *IPUtilPrometheusWrapperImpl) LocalIPs() ([]net.IP, error) {
	// TODO time package conflicts
	start_49a52cf7 := time.Now()
	w.metrics.Total("main", "IPUtil", "LocalIPs")
	a, err := w.wrapped.LocalIPs()
	duration_58a84e00 := time.Since(start_49a52cf7)
	if err != nil {
		w.metrics.Failure("main", "IPUtil", "LocalIPs", duration_58a84e00, err)
		// TODO find a way to add default values here and return the error. for now return the same thing :)
		return a, err
	}

	// TODO if method has no error does success matter or not?
	w.metrics.Success("main", "IPUtil", "LocalIPs", duration_58a84e00)

	return a, err
}
