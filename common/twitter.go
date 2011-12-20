package main

import (
	"fmt"
	"url"
)

type TwStatus struct {
	Created_at string
	Entities   *TwEntities
	User       TwUser
	Place      *TwPlace
	Id         int64
	Text       string
}

type TwEntities struct {
	User_Mentions []*TwUserMention
	Urls          []*TwUrl
	Hashtags      []*TwHashtag
}

func (e *TwEntities) String() string {
	return fmt.Sprintf("{mentions:%s, urls:%s, tags:%s}", e.User_Mentions, e.Urls, e.Hashtags)
}

type TwUserMention struct {
	Screen_Name string
	Id          int64
}

type TwUrl struct {
	Url          string
	Expanded_Url *string
}

func (url *TwUrl) String() string {
	return fmt.Sprintf("{url:%s, ex_url:%s}", url.Url, *(url.Expanded_Url))
}

type TwHashtag struct {
	Text string
}

type TwUser struct {
	Id                int64
	Screen_Name       string
	Profile_Image_Url *string
	UTC_Offset        *int64
}

func (user TwUser) String() string {
	return fmt.Sprintf("{id:%d, sn:%s, piu:%s}", user.Id, user.Screen_Name, *(user.Profile_Image_Url))
}

type TwPlace struct {
	Full_Name    string
	Country_Code string
	Url          string
	Id           string
	Place_Type   string
}

func GetTwitterAccountURL(sn string) string {
	return "http://twitter.com/" + sn
}

func GetSearchResultURL(src string) string {
	return "https://twitter.com/#!/search/" + url.QueryEscape(src)
}

func GetDomainURL(domain string) string {
	return "http://" + domain
}
