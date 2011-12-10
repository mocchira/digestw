package main

import (
	"sort"
	"time"
	"fmt"
//	"log"
)

const (
	ST_TWEET = iota
	ST_PLACE
	ST_HASHTAG
	ST_URL
	ST_DOMAIN
	ST_MENTION
	ST_TWEETER
)

const (
	TU_SUFFIX_TOTAL = "Total"
	TU_SUFFIX_HOUR  = "Hour"
	TU_SUFFIX_DAY   = "Day"
	TU_SUFFIX_WEEK  = "Week"
	TU_SUFFIX_MONTH = "Month"
	TU_KEY_SEP      = "_"
)

var (
	sf        map[string]*Stats
	statsTime *time.Time
)

func init() {
	sf = make(map[string]*Stats)
	statsTime = time.UTC()
}

func Foreach(f func(key string, stats *Stats)) {
	for k, v := range sf {
		f(k, v)
	}
}

func isValidKind(kind int) bool {
	return kind >= ST_TWEET && kind <= ST_TWEETER
}

func SetStatsTime(dt string) {
	statsTime, _ = time.Parse(time.RubyDate, dt)
}

func GetTotalStats(kind int) *Stats {
	if !isValidKind(kind) {
		return nil
	}
	key := fmt.Sprintf("%d_%s", kind, TU_SUFFIX_TOTAL)
	return Get(key)
}

func GetHourStats(kind int) *Stats {
	if !isValidKind(kind) {
		return nil
	}
	key := fmt.Sprintf("%d_%d_%s", kind, statsTime.Hour, TU_SUFFIX_HOUR)
	return Get(key)
}

func GetDayStats(kind int) *Stats {
	if !isValidKind(kind) {
		return nil
	}
	key := fmt.Sprintf("%d_%4d%2d%2d_%s", kind, statsTime.Year, statsTime.Month, statsTime.Day, TU_SUFFIX_DAY)
	return Get(key)
}

func GetWeekStats(kind int) *Stats {
	if !isValidKind(kind) {
		return nil
	}
	key := fmt.Sprintf("%d_%d_%s", kind, statsTime.Weekday, TU_SUFFIX_WEEK)
	return Get(key)
}

func GetMonthStats(kind int) *Stats {
	if !isValidKind(kind) {
		return nil
	}
	key := fmt.Sprintf("%d_%d_%s", kind, statsTime.Month, TU_SUFFIX_MONTH)
	return Get(key)
}

func Get(key string) *Stats {
	if stats, found := sf[key]; found {
		return stats
	}
	sf[key] = newStats()
	return sf[key]
}

type Stats struct {
	Samples     map[string]uint64
	OrderedKeys []string
}

func newStats() *Stats {
	return &Stats{
		Samples:     make(map[string]uint64),
		OrderedKeys: make([]string, 0),
	}
}

func (s *Stats) Get(key string) uint64 {
	return s.Samples[key]
}

func (s *Stats) Keys() []string {
	sort.Sort(s)
	return s.OrderedKeys
}

func (s *Stats) Add(key string) {
	if _, found := s.Samples[key]; !found {
		s.Samples[key] = 0
		s.OrderedKeys = append(s.OrderedKeys, key)
	}
	s.Samples[key]++
}

func (s *Stats) Set(key string, cnt uint64) {
	if _, found := s.Samples[key]; !found {
		s.OrderedKeys = append(s.OrderedKeys, key)
	}
	s.Samples[key] = cnt
}

func (s *Stats) Len() int {
	return len(s.OrderedKeys)
}

func (s *Stats) Less(i int, j int) bool {
	left := s.Samples[s.OrderedKeys[i]]
	right := s.Samples[s.OrderedKeys[j]]
	return left > right
}

func (s *Stats) Swap(i int, j int) {
	s.OrderedKeys[i], s.OrderedKeys[j] = s.OrderedKeys[j], s.OrderedKeys[i]
}
