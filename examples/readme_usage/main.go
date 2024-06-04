package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// TODO find a better way to call with magic comment
// TODO either by putting a binary in PATH or sth else
//
//go:generate go run /home/loop/p/promwrapgen/main.go -m all -t IPUtil
type IPUtil interface {
	PublicIP() (net.IP, error)
	LocalIPs() ([]net.IP, error)
}

type IPUtilImpl struct {
}

func (u *IPUtilImpl) PublicIP() (net.IP, error) {
	if rand.Int()%10 == 0 {
		return nil, errors.New("intentional error")
	}

	resp, err := http.Get("https://api.ipify.org/")
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return net.ParseIP(strings.TrimSpace(string(respBytes))), nil
}

func (u *IPUtilImpl) LocalIPs() ([]net.IP, error) {
	if rand.Int()%10 == 0 {
		return nil, errors.New("intentional error")
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var ips []net.IP
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			return nil, err
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			ips = append(ips, ip)
		}
	}

	return ips, nil
}

type Metrics struct {
	total      *atomic.Int64
	errCnt     *atomic.Int64
	successCnt *atomic.Int64
	durations  []time.Duration
	errs       []error
	mu         *sync.Mutex
}

func (m *Metrics) Total(_ context.Context, name, pkg, intr, method string) {
	fmt.Println("Total", pkg, intr, name)
	m.total.Add(1)
}
func (m *Metrics) Failure(_ context.Context, name, pkg, intr, method string, d time.Duration, err error) {
	fmt.Println("Failure", pkg, intr, method)
	m.errCnt.Add(1)
	m.mu.Lock()
	defer m.mu.Unlock()

	m.errs = append(m.errs, err)
	m.durations = append(m.durations, d)
}
func (m *Metrics) Success(_ context.Context, name, pkg, intr, method string, d time.Duration) {
	fmt.Println("Success", pkg, intr, method)
	m.successCnt.Add(1)
	m.mu.Lock()
	defer m.mu.Unlock()

	m.durations = append(m.durations, d)
}

func (m *Metrics) String() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	dAvg := m.durationAVG()
	errors := m.errors()
	f := `Summary:
total: %d
errCnt: %d
successCnt: %d
durationAvg: %s
errors: %s
`
	return fmt.Sprintf(
		f,
		m.total.Load(),
		m.errCnt.Load(),
		m.successCnt.Load(),
		dAvg.String(),
		errors,
	)

}

func (m *Metrics) durationAVG() time.Duration {
	var t time.Duration
	for _, d := range m.durations {
		t += d
	}

	return time.Duration(int(t.Nanoseconds()) / len(m.durations))
}

func (m *Metrics) errors() string {
	errs := errors.Join(m.errs...)
	if errs != nil {
		return errs.Error()
	}
	return ""
}

func main() {

	m := &Metrics{
		total:      &atomic.Int64{},
		errCnt:     &atomic.Int64{},
		successCnt: &atomic.Int64{},
		durations:  []time.Duration{},
		errs:       []error{},
		mu:         &sync.Mutex{},
	}

	uPromWrapGen := NewIPUtilPrometheusWrapperImpl("iputil", &IPUtilImpl{}, m)

	for i := 0; i < 100; i++ {
		uPromWrapGen.PublicIP()
		uPromWrapGen.LocalIPs()
		time.Sleep(200 * time.Millisecond)
	}

	fmt.Println(m.String())
}
