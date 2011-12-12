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
	"launchpad.net/mgo"
)

var (
	sa   *StatsAll
	sess *mgo.Session
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
	sa.SetStatsTime(twstats.Created_at)
	sa.Total.Tweets.Add(KIND_TWEET, 1)
	hs := sa.GetHourStatsUnit()
	hs.Tweets.Add(KIND_TWEET, 1)
	ds := sa.GetDayStatsUnit()
	ds.Tweets.Add(KIND_TWEET, 1)
	ws := sa.GetWeekStatsUnit()
	ws.Tweets.Add(KIND_TWEET, 1)
	ms := sa.GetMonthStatsUnit()
	ms.Tweets.Add(KIND_TWEET, 1)
	if twstats.Place != nil {
		sa.Total.GetPlacesStats().Add(twstats.Place.Full_Name, 1)
		hs.GetPlacesStats().Add(twstats.Place.Full_Name, 1)
		ds.GetPlacesStats().Add(twstats.Place.Full_Name, 1)
		ws.GetPlacesStats().Add(twstats.Place.Full_Name, 1)
		ms.GetPlacesStats().Add(twstats.Place.Full_Name, 1)
	}
	if twstats.Entities != nil {
		for _, v := range twstats.Entities.User_Mentions {
			sa.Total.GetMentionsStats().Add(v.Screen_Name, 1)
			hs.GetMentionsStats().Add(v.Screen_Name, 1)
			ds.GetMentionsStats().Add(v.Screen_Name, 1)
			ws.GetMentionsStats().Add(v.Screen_Name, 1)
			ms.GetMentionsStats().Add(v.Screen_Name, 1)
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
			sa.Total.GetUrlsStats().Add(orgUrl.Raw, 1)
			hs.GetUrlsStats().Add(orgUrl.Raw, 1)
			ds.GetUrlsStats().Add(orgUrl.Raw, 1)
			ws.GetUrlsStats().Add(orgUrl.Raw, 1)
			ms.GetUrlsStats().Add(orgUrl.Raw, 1)

			sa.Total.GetDomainsStats().Add(orgUrl.Host, 1)
			hs.GetDomainsStats().Add(orgUrl.Host, 1)
			ds.GetDomainsStats().Add(orgUrl.Host, 1)
			ws.GetDomainsStats().Add(orgUrl.Host, 1)
			ms.GetDomainsStats().Add(orgUrl.Host, 1)
		}
		for _, v := range twstats.Entities.Hashtags {
			sa.Total.GetHashtagsStats().Add(v.Text, 1)
			hs.GetHashtagsStats().Add(v.Text, 1)
			ds.GetHashtagsStats().Add(v.Text, 1)
			ws.GetHashtagsStats().Add(v.Text, 1)
			ms.GetHashtagsStats().Add(v.Text, 1)
		}
	}
	sa.Total.GetTweetersStats().Add(twstats.User.Screen_Name, 1)
	hs.GetTweetersStats().Add(twstats.User.Screen_Name, 1)
	ds.GetTweetersStats().Add(twstats.User.Screen_Name, 1)
	ws.GetTweetersStats().Add(twstats.User.Screen_Name, 1)
	ms.GetTweetersStats().Add(twstats.User.Screen_Name, 1)
}

func filter(kind, unit string, stats *Stats) {
	log.Printf("kind: %s unit:%s", kind, unit)
	keys := stats.Keys()
	for i := 0; i < len(keys); i++ {
		log.Printf("rank:%2d key:%s count:%d", i+1, keys[i], stats.Get(keys[i]))
	}
}

func update(col string, su *StatsUnit) {
	var nsu StatsUnit
	var err os.Error
	if err = nsu.Find(sess, col, su.UserId, su.UnitId); err != nil && err != mgo.NotFound {
		panic(err)
	}
	if err != mgo.NotFound {
		su.Add(&nsu)
	}
	su.ForeachStats(filter)
	if _, err = su.Upsert(sess, col, su.UserId, su.UnitId); err != nil {
		panic(err)
	}
	log.Printf("result %v", su)
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
		sa = NewStatsAll("mocchira")
		tl := decode()
		for _, v := range tl {
			addStats(&v)
		}
		var err os.Error
		sess, err = mgo.Mongo("localhost")
		if err != nil {
			panic(err)
		}
		defer sess.Close()
		sa.Foreach(update)
		//sa.ForeachStats(output)
		return
	} else {
		// mongotest
		/*
			sc := NewStatsUnit("mocchira", "23")
			sc.Places.Set("Tokyo", 2)
			sc.Places.Set("Yokohama", 999)
			sc.Mentions.Set("bikki", 999)
			if err = sc.Insert(session, MGO_COL_STATS_HOUR); err != nil {
				panic(err)
			}
			nsc := NewStatsCol("", "")
			if err = nsc.Find(session, MGO_COL_STATS_HOUR, "mocchira", ""); err != nil {
				panic(err)
			}
			log.Printf("result %v", nsc)
		*/
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
