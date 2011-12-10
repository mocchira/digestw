package main

import (
	"http"
	"url"
	"os"
)

// GetFinalURL resolves a URL to a URL which represents a final location url
func GetFinalURL(url string) (*url.URL, os.Error) {
	res, err := http.Head(url)
	if err != nil {
		return nil, err
	}
	return res.Request.URL, nil
}

