package main

import (
	"os"
	"url"
	"log"
	"strconv"
	"time"
	"github.com/mrjones/oauth"
	"launchpad.net/mgo"
)

func addStats(sa *StatsAll, twstats *TwStatus, resolveURL bool) {
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
		if resolveURL {
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

func update(sess *mgo.Session, col string, su *StatsUnit) {
	var nsu StatsUnit
	var err os.Error
	if err = nsu.Find(sess, col, su.UserId, su.UnitId); err != nil && err != mgo.NotFound {
		log.Print(err)
		return
	}
	if err != mgo.NotFound {
		su.Add(&nsu)
	}
	su.ForeachStats(func(kind, unit string, stats *Stats) { stats.Keys() })
	if _, err = su.Upsert(sess, col, su.UserId, su.UnitId); err != nil {
		log.Print(err)
		return
	}
}

func RegistUser(sess *mgo.Session, tu *TwUser, at *oauth.AccessToken) (*DigestwUser, os.Error) {
	du := NewDigestwUser(tu, at)
	if _, err := du.Upsert(sess); err != nil {
		return nil, err
	}
	return du, nil
}

func Crawl(pool *mgo.Session, du *DigestwUser, tl *TwTimeLine, resolveURL bool, done chan<- int) {
	defer func() { done <- 0 }()
	sess := pool.New()
	defer sess.Close()
	sa := NewStatsAll(du.TwUser.Screen_Name, sess)
	var sid, first, last int64
	for k, v := range *tl {
		tmpTime, _ := time.Parse(time.RubyDate, v.Created_at)
		tmpSec := tmpTime.Seconds()
		if du.UTC_Offset != nil {
			tmpSec += *du.UTC_Offset
			v.Created_at = time.SecondsToUTC(tmpSec).Format(time.RubyDate)
		}
		if k == 0 {
			sid = v.Id
			last = tmpSec
		}
		addStats(sa, &v, resolveURL)
		if k == (len(*tl) - 1) {
			first = tmpSec
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
}
