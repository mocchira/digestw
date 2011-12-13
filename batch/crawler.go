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
	"io"
	"log"
	"strconv"
	"json"
	"time"
	"github.com/mrjones/oauth"
	"launchpad.net/mgo"
)

const (
	MODE_TEST       = "test"
	MODE_INIT_OAUTH = "oauth"
	MODE_DEFAULT    = "default"
)

var (
	sess *mgo.Session
)

func decode(r io.Reader) []TwStatus {
	var tl []TwStatus
	dec := json.NewDecoder(r)
	if err := dec.Decode(&tl); err != nil {
		log.Fatal(err)
	}
	return tl
}

func addStats(sa *StatsAll, twstats *TwStatus) {
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
		log.Print(err)
		return
	}
	if err != mgo.NotFound {
		su.Add(&nsu)
	}
	su.ForeachStats(filter)
	if _, err = su.Upsert(sess, col, su.UserId, su.UnitId); err != nil {
		log.Print(err)
		return
	}
	log.Printf("result %v", su)
}

func registUser(r io.Reader, at *oauth.AccessToken) *DigestwUser {
	var tu TwUser
	dec := json.NewDecoder(r)
	if err := dec.Decode(&tu); err != nil {
		log.Fatal(err)
	}
	du := NewDigestwUser(&tu, at)
	if _, err := du.Upsert(sess); err != nil {
		log.Fatal(err)
	}
	return du
}

func crawl(c *oauth.Consumer, du *DigestwUser, r io.Reader, count int, done chan<- int) {
	sa := NewStatsAll(du.TwUser.Screen_Name)
	params := map[string]string{"include_entities": "true", "count": strconv.Itoa(count)}
	if r == nil {
		if du.SinceId != "" {
			params["since_id"] = du.SinceId
		}
		response, err := c.Get(
			"https://api.twitter.com/1/statuses/home_timeline.json",
			params,
			&(du.AccessToken))
		if err != nil {
			log.Print(err)
			done <- 0
			return
		}
		defer response.Body.Close()
		r = response.Body
	}
	tl := decode(r)
	var sid, first, last int64
	for k, v := range tl {
		if k == 0 {
			sid = v.Id
			tmp, _ := time.Parse(time.RubyDate, v.Created_at)
			last = tmp.Seconds()
		}
		addStats(sa, &v)
		if k == (len(tl) - 1) {
			tmp, _ := time.Parse(time.RubyDate, v.Created_at)
			first = tmp.Seconds()
		}
		log.Printf("id:%d", v.Id)
	}
	sa.Foreach(update)
	// decide to the next execution time
	du.SinceId = strconv.Itoa64(sid)
	du.NextSeconds = time.Seconds() + (last - first)
	if _, err := du.Upsert(sess); err != nil {
		log.Print(err)
	}
	done <- 0
}

func main() {
	var consumerKey *string = flag.String("consumerkey", "RMA3YnQen7J0SDX67b5g", "")
	var consumerSecret *string = flag.String("consumersecret", "87GYFCqZz2k9VLcatBp7cpajzcdxRRPKfa3pMPtgW4", "")
	var count *int = flag.Int("count", 100, "")
	var mode *string = flag.String("mode", "default", "")

	flag.Parse()

	// init
	c := oauth.NewConsumer(
		*consumerKey,
		*consumerSecret,
		oauth.ServiceProvider{
			RequestTokenUrl:   "https://api.twitter.com/oauth/request_token",
			AuthorizeTokenUrl: "https://api.twitter.com/oauth/authorize",
			AccessTokenUrl:    "https://api.twitter.com/oauth/access_token",
		})
	c.Debug(true)
	var err os.Error
	sess, err = mgo.Mongo("localhost")
	if err != nil {
		panic(err)
	}
	defer sess.Close()

	switch *mode {
	case MODE_TEST:
		// js test
		var du DigestwUser
		du.TwUser.Screen_Name = "mocchira"
		done := make(chan int)
		go crawl(c, &du, os.Stdin, *count, done)
		<-done
		return
	case MODE_INIT_OAUTH:
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
			"https://api.twitter.com/1/account/verify_credentials.json",
			map[string]string{"skip_status": "true"},
			accessToken)
		if err != nil {
			log.Fatal(err)
		}
		defer response.Body.Close()
		du := registUser(response.Body, accessToken)
		fmt.Println("id:" + du.TwUser.Screen_Name)
		return
	default:
		idx := 0
		dulist := [CRAWL_UNIT]DigestwUser{}
		done := make(chan int)
		iter := dulist[0].Find(sess, time.Seconds())
		for iter.Next(&dulist[idx]) {
			go crawl(c, &dulist[idx], nil, *count, done)
			idx++
		}
		for ; idx > 0; idx-- {
			<-done
		}
		if iter.Err() != nil {
			log.Fatal(err)
		}
		return
	}

}
