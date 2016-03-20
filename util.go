package main

import (
	"net"
	"net/http"
	"strings"
	"time"
)

func getResponseTimes(req *http.Request) (resp *http.Response, err error) {
	for i := 0; i < retryTimes; i++ {
		var trans http.Transport
		resp, err = trans.RoundTrip(req)
		if err != nil {
			continue
		}
		break
	}
	return
}

func dialTimeoutTimes(network, address string, timeout time.Duration, times int) (c net.Conn, err error) {
	if !strings.Contains(address, ":") {
		address += ":80"
	}

	for i := 0; i < times; i++ {
		c, err = net.DialTimeout(network, address, timeout)
		if err != nil {
			continue
		}
		return
	}
	return
}
