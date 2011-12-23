package main

import (
	"http"
	"url"
	"os"
	"net"
)

const (
	HTTP_TIMEOUT = 1e9 * 10 // 10sec
)

var (
	DigestHttpClient = &http.Client{Transport: &http.Transport{Dial: timeoutDialler(HTTP_TIMEOUT)}}
)

func timeoutDialler(ns int64) func(net, addr string) (c net.Conn, err os.Error) {
	return func(netw, addr string) (net.Conn, os.Error) {
		c, err := net.Dial(netw, addr)
		if err != nil {
			return nil, err
		}
		c.SetTimeout(ns)
		return c, nil
	}
}

// GetFinalURL resolves a URL to a URL which represents a final location url
func GetFinalURL(url string) (*url.URL, os.Error) {
	res, err := DigestHttpClient.Head(url)
	if err != nil {
		return nil, err
	}
	return res.Request.URL, nil
}
