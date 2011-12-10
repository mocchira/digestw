// Copyright (c) 2010 The GOAuth Authors. All rights reserved.
//
//     email - hoka@hokapoka.com
//       web - http://go.hokapoka.com
//      buzz - hokapoka.com@gmail.com 
//   twitter - @hokapokadotcom
//    github - github.com/hokapoka/goauth
//   
package main

import (
	"os"
	"url"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"json"
	"github.com/mrjones/oauth"
)

func decode() []TwStatus {
	var tl []TwStatus
	dec := json.NewDecoder(os.Stdin)
	if err := dec.Decode(&tl); err != nil {
		log.Fatal(err)
	}
	return tl
}

func addStats(twstats *TwStatus) {
	SetStatsTime(twstats.Created_at)
	pstats := GetTotalStats(ST_TWEET)
	pstats.Add(TU_SUFFIX_TOTAL)
	pstats = GetHourStats(ST_TWEET)
	pstats.Add(TU_SUFFIX_TOTAL)
	pstats = GetDayStats(ST_TWEET)
	pstats.Add(TU_SUFFIX_TOTAL)
	pstats = GetWeekStats(ST_TWEET)
	pstats.Add(TU_SUFFIX_TOTAL)
	pstats = GetMonthStats(ST_TWEET)
	pstats.Add(TU_SUFFIX_TOTAL)
	if twstats.Place != nil {
		pstats = GetTotalStats(ST_PLACE)
		pstats.Add(twstats.Place.Full_Name)
		pstats = GetHourStats(ST_PLACE)
		pstats.Add(twstats.Place.Full_Name)
		pstats = GetDayStats(ST_PLACE)
		pstats.Add(twstats.Place.Full_Name)
		pstats = GetWeekStats(ST_PLACE)
		pstats.Add(twstats.Place.Full_Name)
		pstats = GetMonthStats(ST_PLACE)
		pstats.Add(twstats.Place.Full_Name)
	}
	if twstats.Entities != nil {
		for _, v := range twstats.Entities.User_Mentions {
			pstats = GetTotalStats(ST_MENTION)
			pstats.Add(v.Screen_Name)
			pstats = GetHourStats(ST_MENTION)
			pstats.Add(v.Screen_Name)
			pstats = GetDayStats(ST_MENTION)
			pstats.Add(v.Screen_Name)
			pstats = GetWeekStats(ST_MENTION)
			pstats.Add(v.Screen_Name)
			pstats = GetMonthStats(ST_MENTION)
			pstats.Add(v.Screen_Name)
		}
		for _, v := range twstats.Entities.Urls {
			var orgUrl *url.URL
			var err os.Error
			if v.Expanded_Url == nil {
				if orgUrl, err = GetFinalURL(v.Url); err != nil {
					continue
				}
			} else {
				if orgUrl, err = GetFinalURL(*v.Expanded_Url); err != nil {
					continue
				}
			}
			pstats = GetTotalStats(ST_URL)
			pstats.Add(orgUrl.Raw)
			pstats = GetHourStats(ST_URL)
			pstats.Add(orgUrl.Raw)
			pstats = GetDayStats(ST_URL)
			pstats.Add(orgUrl.Raw)
			pstats = GetWeekStats(ST_URL)
			pstats.Add(orgUrl.Raw)
			pstats = GetMonthStats(ST_URL)
			pstats.Add(orgUrl.Raw)

			pstats = GetTotalStats(ST_DOMAIN)
			pstats.Add(orgUrl.Host)
			pstats = GetHourStats(ST_DOMAIN)
			pstats.Add(orgUrl.Host)
			pstats = GetDayStats(ST_DOMAIN)
			pstats.Add(orgUrl.Host)
			pstats = GetWeekStats(ST_DOMAIN)
			pstats.Add(orgUrl.Host)
			pstats = GetMonthStats(ST_DOMAIN)
			pstats.Add(orgUrl.Host)
		}
		for _, v := range twstats.Entities.Hashtags {
			pstats = GetTotalStats(ST_HASHTAG)
			pstats.Add(v.Text)
			pstats = GetHourStats(ST_HASHTAG)
			pstats.Add(v.Text)
			pstats = GetDayStats(ST_HASHTAG)
			pstats.Add(v.Text)
			pstats = GetWeekStats(ST_HASHTAG)
			pstats.Add(v.Text)
			pstats = GetMonthStats(ST_HASHTAG)
			pstats.Add(v.Text)
		}
	}
	pstats = GetTotalStats(ST_TWEETER)
	pstats.Add(twstats.User.Screen_Name)
	pstats = GetHourStats(ST_TWEETER)
	pstats.Add(twstats.User.Screen_Name)
	pstats = GetDayStats(ST_TWEETER)
	pstats.Add(twstats.User.Screen_Name)
	pstats = GetWeekStats(ST_TWEETER)
	pstats.Add(twstats.User.Screen_Name)
	pstats = GetMonthStats(ST_TWEETER)
	pstats.Add(twstats.User.Screen_Name)
}

func output(key string, stats *Stats) {
	log.Printf("key: %s", key)
	keys := stats.Keys()
	for i := 0; i < len(keys); i++ {
		log.Printf("rank:%2d key:%s count:%d", i+1, keys[i], stats.Get(keys[i]))
	}
}

func main() {
	var consumerKey *string = flag.String("consumerkey", "RMA3YnQen7J0SDX67b5g", "")
	var consumerSecret *string = flag.String("consumersecret", "87GYFCqZz2k9VLcatBp7cpajzcdxRRPKfa3pMPtgW4", "")
	var count *int = flag.Int("count", 100, "")
	var since_id *string = flag.String("since_id", "143280351165415425", "")
	var jsmode *bool = flag.Bool("js", true, "")

	flag.Parse()
	if *jsmode {
		// js test
		tl := decode()
		for _, v := range tl {
			addStats(&v)
		}
		Foreach(output)
		return
	} else {
		// http test
		urls := [...]string{
			"http://t.co/HPv2ieNu",
			"http://t.co/3KhOs391",
			"http://t.co/Re95sQbh",
		}
		for _, v := range urls {
			ret, err := GetFinalURL(v)
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("src:%s dst:%s\n", v, ret)
		}
		return
	}

	c := oauth.NewConsumer(
		*consumerKey,
		*consumerSecret,
		oauth.ServiceProvider{
			RequestTokenUrl:   "https://api.twitter.com/oauth/request_token",
			AuthorizeTokenUrl: "https://api.twitter.com/oauth/authorize",
			AccessTokenUrl:    "https://api.twitter.com/oauth/access_token",
		})
	c.Debug(true)
	//requestToken, url, err := c.GetRequestTokenAndUrl("http://www.mocchira.com/")
	requestToken, url, err := c.GetRequestTokenAndUrl("oob")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("(1) Go to: " + url)
	fmt.Println("(2) Grant access, you should get back a verification code.")
	fmt.Println("(3) Enter that verification code here: ")
	verificationCode := ""
	fmt.Scanln(&verificationCode)

	accessToken, err := c.AuthorizeToken(requestToken, verificationCode)
	if err != nil {
		log.Fatal(err)
	}
	response, err := c.Get(
		"https://api.twitter.com/1/statuses/home_timeline.json",
		map[string]string{"include_entities": "true", "count": strconv.Itoa(*count), "since_id": *since_id},
		accessToken)
	defer response.Body.Close()
	bits, err := ioutil.ReadAll(response.Body)
	fmt.Println("Twitter RESPONSE: " + string(bits))

}
