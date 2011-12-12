package main

import (
	"os"
	"sort"
	"time"
	"fmt"
	"strconv"
	"launchpad.net/gobson/bson"
	"launchpad.net/mgo"
	//	"log"
)

const (
	KIND_TWEET   = "Tweet"
	KIND_PLACE   = "Place"
	KIND_HASHTAG = "Hashtag"
	KIND_URL     = "Url"
	KIND_DOMAIN  = "Domain"
	KIND_MENTION = "Mentioner"
	KIND_TWEETER = "Tweeter"
)

const (
	MGO_DB              = "digestw"
	MGO_COL_STATS_TOTAL = "stats_total"
	MGO_COL_STATS_HOUR  = "stats_hour"
	MGO_COL_STATS_DAY   = "stats_day"
	MGO_COL_STATS_WEEK  = "stats_week"
	MGO_COL_STATS_MONTH = "stats_month"
)

const (
	REC_MAX_COUNT = 30
)

type Stats struct {
	Samples     map[string]uint64
	orderedKeys []string
}

func newStats() *Stats {
	return &Stats{
		Samples:     make(map[string]uint64),
		orderedKeys: make([]string, 0),
	}
}

func (s *Stats) Get(key string) uint64 {
	return s.Samples[key]
}

func (s *Stats) Keys() []string {
	sort.Sort(s)
	if len(s.orderedKeys) > REC_MAX_COUNT {
		delKeys := s.orderedKeys[REC_MAX_COUNT:]
		for _, key := range delKeys {
			s.Samples[key] = 0, false
		}
		s.orderedKeys = s.orderedKeys[:REC_MAX_COUNT]
	}
	return s.orderedKeys
}

func (s *Stats) Add(key string, add uint64) {
	if _, found := s.Samples[key]; !found {
		s.Samples[key] = 0
		s.orderedKeys = append(s.orderedKeys, key)
	}
	s.Samples[key] += add
}

func (s *Stats) AddStats(src *Stats) *Stats {
	if s == nil {
		return src
	}
	if src == nil {
		return s
	}
	for k, v := range src.Samples {
		s.Add(k, v)
	}
	return s
}

func (s *Stats) Set(key string, cnt uint64) {
	if _, found := s.Samples[key]; !found {
		s.orderedKeys = append(s.orderedKeys, key)
	}
	s.Samples[key] = cnt
}

func (s *Stats) Len() int {
	return len(s.orderedKeys)
}

func (s *Stats) Less(i int, j int) bool {
	left := s.Samples[s.orderedKeys[i]]
	right := s.Samples[s.orderedKeys[j]]
	return left > right
}

func (s *Stats) Swap(i int, j int) {
	s.orderedKeys[i], s.orderedKeys[j] = s.orderedKeys[j], s.orderedKeys[i]
}

func (s *Stats) String() string {
	return fmt.Sprintf("%v", s.Samples)
}

type StatsUnit struct {
	UserId   string // compound key
	UnitId   string // compound key
	Tweets   *Stats ",omitempty"
	Places   *Stats ",omitempty"
	Hashtags *Stats ",omitempty"
	Urls     *Stats ",omitempty"
	Domains  *Stats ",omitempty"
	Mentions *Stats ",omitempty"
	Tweeters *Stats ",omitempty"
}

func NewStatsUnit(uid, unitid string) *StatsUnit {
	return &StatsUnit{
		UserId: uid,
		UnitId: unitid,
		Tweets: newStats(),
	}
}

func (su *StatsUnit) String() string {
	return fmt.Sprintf("{uid:%s unit:%s tw:%s pl:%s hash:%s url:%s dm:%s me:%s twter:%s}",
		su.UserId,
		su.UnitId,
		su.Tweets,
		su.Places,
		su.Hashtags,
		su.Urls,
		su.Domains,
		su.Mentions,
		su.Tweeters)
}

func (su *StatsUnit) Add(src *StatsUnit) *StatsUnit {
	if src == nil {
		return su
	}
	if su == nil {
		return src
	}
	su.Tweets = su.Tweets.AddStats(src.Tweets)
	su.Places = su.Places.AddStats(src.Places)
	su.Hashtags = su.Hashtags.AddStats(src.Hashtags)
	su.Urls = su.Urls.AddStats(src.Urls)
	su.Domains = su.Domains.AddStats(src.Domains)
	su.Mentions = su.Mentions.AddStats(src.Mentions)
	su.Tweeters = su.Tweeters.AddStats(src.Tweeters)
	return su
}

func (su *StatsUnit) GetPlacesStats() *Stats {
	if su.Places == nil {
		su.Places = newStats()
	}
	return su.Places
}

func (su *StatsUnit) GetHashtagsStats() *Stats {
	if su.Hashtags == nil {
		su.Hashtags = newStats()
	}
	return su.Hashtags
}

func (su *StatsUnit) GetUrlsStats() *Stats {
	if su.Urls == nil {
		su.Urls = newStats()
	}
	return su.Urls
}

func (su *StatsUnit) GetDomainsStats() *Stats {
	if su.Domains == nil {
		su.Domains = newStats()
	}
	return su.Domains
}

func (su *StatsUnit) GetMentionsStats() *Stats {
	if su.Mentions == nil {
		su.Mentions = newStats()
	}
	return su.Mentions
}

func (su *StatsUnit) GetTweetersStats() *Stats {
	if su.Tweeters == nil {
		su.Tweeters = newStats()
	}
	return su.Tweeters
}

func (su *StatsUnit) Upsert(sess *mgo.Session, col, uid, unitid string) (interface{}, os.Error) {
	c := sess.DB(MGO_DB).C(col)
	return c.Upsert(bson.M{"userid": uid, "unitid": unitid}, su)
}

func (su *StatsUnit) Find(sess *mgo.Session, col, uid, unitid string) os.Error {
	c := sess.DB(MGO_DB).C(col)
	if unitid == "" {
		return c.Find(bson.M{"userid": uid}).One(su)
	}
	return c.Find(bson.M{"userid": uid, "unitid": unitid}).One(su)
}

type StatsAll struct {
	Total     *StatsUnit
	hour      []*StatsUnit
	day       []*StatsUnit
	week      []*StatsUnit
	month     []*StatsUnit
	statsTime *time.Time
	UserId    string
}

func NewStatsAll(uid string) *StatsAll {
	return &StatsAll{
		Total:     NewStatsUnit(uid, ""),
		hour:      make([]*StatsUnit, 0),
		day:       make([]*StatsUnit, 0),
		week:      make([]*StatsUnit, 0),
		month:     make([]*StatsUnit, 0),
		statsTime: time.UTC(),
		UserId:    uid,
	}
}

func (su *StatsUnit) ForeachStats(f func(kind, unit string, stats *Stats)) {
	f(KIND_TWEET, su.UnitId, su.Tweets)
	if su.Places != nil {
		f(KIND_PLACE, su.UnitId, su.Places)
	}
	if su.Hashtags != nil {
		f(KIND_HASHTAG, su.UnitId, su.Hashtags)
	}
	if su.Urls != nil {
		f(KIND_URL, su.UnitId, su.Urls)
	}
	if su.Domains != nil {
		f(KIND_DOMAIN, su.UnitId, su.Domains)
	}
	if su.Mentions != nil {
		f(KIND_MENTION, su.UnitId, su.Mentions)
	}
	if su.Tweeters != nil {
		f(KIND_TWEETER, su.UnitId, su.Tweeters)
	}
}

func (sa *StatsAll) ForeachStats(f func(kind, unit string, stats *Stats)) {
	sa.Total.ForeachStats(f)
	for _, v := range sa.hour {
		v.ForeachStats(f)
	}
	for _, v := range sa.day {
		v.ForeachStats(f)
	}
	for _, v := range sa.week {
		v.ForeachStats(f)
	}
	for _, v := range sa.month {
		v.ForeachStats(f)
	}
}

func (sa *StatsAll) Foreach(f func(col string, su *StatsUnit)) {
	f(MGO_COL_STATS_TOTAL, sa.Total)
	for _, v := range sa.hour {
		f(MGO_COL_STATS_HOUR, v)
	}
	for _, v := range sa.day {
		f(MGO_COL_STATS_DAY, v)
	}
	for _, v := range sa.week {
		f(MGO_COL_STATS_WEEK, v)
	}
	for _, v := range sa.month {
		f(MGO_COL_STATS_MONTH, v)
	}
}

func (sa *StatsAll) SetStatsTime(dt string) {
	sa.statsTime, _ = time.Parse(time.RubyDate, dt)
}

func (sa *StatsAll) GetHourStatsUnit() *StatsUnit {
	key := strconv.Itoa(sa.statsTime.Hour)
	for _, v := range sa.hour {
		if v.UnitId == key {
			return v
		}
	}
	su := NewStatsUnit(sa.UserId, key)
	sa.hour = append(sa.hour, su)
	return su
}

func (sa *StatsAll) GetDayStatsUnit() *StatsUnit {
	key := fmt.Sprintf("%4d%2d%2d", sa.statsTime.Year, sa.statsTime.Month, sa.statsTime.Day)
	for _, v := range sa.day {
		if v.UnitId == key {
			return v
		}
	}
	su := NewStatsUnit(sa.UserId, key)
	sa.day = append(sa.day, su)
	return su
}

func (sa *StatsAll) GetWeekStatsUnit() *StatsUnit {
	key := strconv.Itoa(sa.statsTime.Weekday)
	for _, v := range sa.week {
		if v.UnitId == key {
			return v
		}
	}
	su := NewStatsUnit(sa.UserId, key)
	sa.week = append(sa.week, su)
	return su
}

func (sa *StatsAll) GetMonthStatsUnit() *StatsUnit {
	key := strconv.Itoa(sa.statsTime.Month)
	for _, v := range sa.month {
		if v.UnitId == key {
			return v
		}
	}
	su := NewStatsUnit(sa.UserId, key)
	sa.month = append(sa.month, su)
	return su
}
