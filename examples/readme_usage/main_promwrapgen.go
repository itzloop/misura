// This code is generate by github.com/itzloop/promwrapgen. DO NOT EDIT!
package main

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

// IPUtilPrometheusWrapperImpl wraps IPUtil and adds metrics like:
// 1. success count
// 2. error count
// 3. total count
// 4. duration
type IPUtilPrometheusWrapperImpl struct {
	// TODO what are fields are required
	intr    string
	wrapped IPUtil
	metrics interface {
		// Failure will be called when err != nil passing the duration and err to it
		Failure(ctx context.Context, pkg, intr, method string, duration time.Duration, err error)

		// Success will be called if err == nil passing the duration to it
		Success(ctx context.Context, pkg, intr, method string, duration time.Duration)

		// Total will be called as soon as the function is called.
		Total(ctx context.Context, pkg, intr, method string)
	}
}

func NewIPUtilPrometheusWrapperImpl(
	wrapped IPUtil,
	metrics interface {
		Failure(ctx context.Context, pkg, intr, method string, duration time.Duration, err error)
		Success(ctx context.Context, pkg, intr, method string, duration time.Duration)
		Total(ctx context.Context, pkg, intr, method string)
	},
) *IPUtilPrometheusWrapperImpl {
	var intr string
	splited := strings.Split(fmt.Sprintf("%T", wrapped), ".")
	if len(splited) != 2 {
		intr = "IPUtil"
	} else {
		intr = splited[1]
	}

	return &IPUtilPrometheusWrapperImpl{
		intr:    intr,
		wrapped: wrapped,
		metrics: metrics,
	}
}

// PublicIP wraps another instance of IPUtil and
// adds prometheus metrics. See PublicIP on IPUtilPrometheusWrapperImpl.wrapped for
// more information.
func (w *IPUtilPrometheusWrapperImpl) PublicIP() (net.IP, error) {
	// TODO time package conflicts
	startF48DC86F := time.Now()
	w.metrics.Total(context.Background(), "main", w.intr, "PublicIP")
	a, err := w.wrapped.PublicIP()
	durationF48DC86F := time.Since(startF48DC86F)
	if err != nil {
		w.metrics.Failure(context.Background(), "main", w.intr, "PublicIP", durationF48DC86F, err)
		// TODO find a way to add default values here and return the error. for now return the same thing :)
		return a, err
	}

	// TODO if method has no error does success matter or not?
	w.metrics.Success(context.Background(), "main", w.intr, "PublicIP", durationF48DC86F)

	return a, err
}

// LocalIPs wraps another instance of IPUtil and
// adds prometheus metrics. See LocalIPs on IPUtilPrometheusWrapperImpl.wrapped for
// more information.
func (w *IPUtilPrometheusWrapperImpl) LocalIPs() ([]net.IP, error) {
	// TODO time package conflicts
	startF48DC86F := time.Now()
	w.metrics.Total(context.Background(), "main", w.intr, "LocalIPs")
	a, err := w.wrapped.LocalIPs()
	durationF48DC86F := time.Since(startF48DC86F)
	if err != nil {
		w.metrics.Failure(context.Background(), "main", w.intr, "LocalIPs", durationF48DC86F, err)
		// TODO find a way to add default values here and return the error. for now return the same thing :)
		return a, err
	}

	// TODO if method has no error does success matter or not?
	w.metrics.Success(context.Background(), "main", w.intr, "LocalIPs", durationF48DC86F)

	return a, err
}
