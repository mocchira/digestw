package main

import (
	"utf8"
)

func CutString(src, suffix string, length int) string {
	utfSrc := utf8.NewString(src)
	utfSfx := utf8.NewString(suffix)
	srcLen := utfSrc.RuneCount()
	sfxLen := utfSfx.RuneCount()
	if srcLen < length || length < sfxLen {
		//nothing to do
		return src
	}
	return utfSrc.Slice(0, length-sfxLen) + suffix
}

func GenAnchorTagStr(disp, url string) string {
	return `<a href="` + url + `">` + disp + `</a>`
}
