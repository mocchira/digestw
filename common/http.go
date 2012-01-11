package main

import (
	"net"
	"net/http"
	"net/url"
)

const (
	HTTP_TIMEOUT = 1e9 * 5 // 5sec
)

var (
	// DigestHttpClient is used by GetFinalURL
	DigestHttpClient = &http.Client{
		Transport: &http.Transport{
			Dial:                timeoutDialler(HTTP_TIMEOUT),
			MaxIdleConnsPerHost: CRAWL_UNIT, // because each of crawling goroutines will access to specific url shorter providers
		},
	}
)

func timeoutDialler(ns int64) func(net, addr string) (c net.Conn, err error) {
	return func(netw, addr string) (net.Conn, error) {
		c, err := net.Dial(netw, addr)
		if err != nil {
			return nil, err
		}
		c.SetTimeout(ns)
		return c, nil
	}
}

// GetFinalURL resolves a URL to a URL which represents a final location
func GetFinalURL(url string) (*url.URL, error) {
	res, err := DigestHttpClient.Head(url)
	if err != nil {
		return nil, err
	}
	return res.Request.URL, nil
}
