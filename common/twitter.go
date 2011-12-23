package main

import (
	"fmt"
	"strconv"
	"json"
	"io"
	"os"
	"github.com/mrjones/oauth"
)

const (
	TW_URL_VERIFY_CREDENTIAL = "https://api.twitter.com/1/account/verify_credentials.json"
	TW_URL_HOME_TIMELINE     = "https://api.twitter.com/1/statuses/home_timeline.json"
)

type TwStatus struct {
	Created_at string
	Entities   *TwEntities
	User       TwUser
	Place      *TwPlace
	Id         int64
	Text       string
}

type TwTimeLine []TwStatus

func (tl *TwTimeLine) Get(r io.Reader) os.Error {
	dec := json.NewDecoder(r)
	if err := dec.Decode(tl); err != nil {
		return err
	}
	return nil
}

func (tl *TwTimeLine) GetFromAPI(c *oauth.Consumer, at *oauth.AccessToken, count int, sinceId string) os.Error {
	params := map[string]string{"include_entities": "true", "count": strconv.Itoa(count)}
	if sinceId != "" && sinceId != "0" {
		params["since_id"] = sinceId
	}
	response, err := c.Get(
		TW_URL_HOME_TIMELINE,
		params,
		at)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return tl.Get(response.Body)
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

func (user *TwUser) Get(r io.Reader) os.Error {
	dec := json.NewDecoder(r)
	if err := dec.Decode(user); err != nil {
		return err
	}
	return nil
}

func (user *TwUser) GetFromAPI(c *oauth.Consumer, at *oauth.AccessToken) os.Error {
	response, err := c.Get(
		TW_URL_VERIFY_CREDENTIAL,
		map[string]string{"skip_status": "true"},
		at)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return user.Get(response.Body)
}

func (user *TwUser) String() string {
	return fmt.Sprintf("{id:%d, sn:%s, piu:%s}", user.Id, user.Screen_Name, *(user.Profile_Image_Url))
}

type TwPlace struct {
	Full_Name    string
	Country_Code string
	Url          string
	Id           string
	Place_Type   string
}
