package main

import (
	"os"
	"io"
	"log"
	"fmt"
	"flag"
	"template"
	"strconv"
	"strings"
	"json"
	"http"
	"url"
	"time"
	"github.com/hoisie/web.go"
	"launchpad.net/mgo"
	"github.com/mrjones/oauth"
)

const (
	HOME_URL        = "http://digestw.stoic.co.jp/web/stats/mocchira/total/"
	CALLBACK_URL    = "http://digestw.stoic.co.jp/web/callback"
	ERR_MSG         = "Server Error"
	STYLE_CLASS_NON = "non"
	STYLE_CLASS_SEL = "selected"
)

var (
	consumer *oauth.Consumer
	mgoPool  *mgo.Session
	tplSet   *template.Set
	unit2col = map[string]string{
		"total": MGO_COL_STATS_TOTAL,
		"month": MGO_COL_STATS_MONTH,
		"week":  MGO_COL_STATS_WEEK,
		"day":   MGO_COL_STATS_DAY,
		"hour":  MGO_COL_STATS_HOUR,
	}
	unit2def = map[string]string{
		"total": "",
		"month": "12",
		"week":  "0",
		"day":   "20111219",
		"hour":  "12",
	}
	unit2linkfun = map[string]func(string, string) []*UnitStyle{
		"total": nil,
		"month": genMonthLinks,
		"week":  genWeekLinks,
		"day":   genDayLinks,
		"hour":  genHourLinks,
	}
	WEEK_LIST = [...]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
)

type UnitStyle struct {
	Class string
	Name  string
}
type Beans struct {
	Data  *StatsUnit
	Units []*UnitStyle
	Links []*UnitStyle
}

func genMonthLinks(uid, val string) []*UnitStyle {
	ret := make([]*UnitStyle, 0)
	for i := 1; i <= 12; i++ {
		m := strconv.Itoa(i)
		var ul *UnitStyle
		if val == m {
			ul = &UnitStyle{
				Name:  m,
				Class: STYLE_CLASS_SEL,
			}
		} else {
			ul = &UnitStyle{
				Name: GenAnchorTagStr(m, "/web/stats/"+uid+"/month/"+m),
			}
		}
		ret = append(ret, ul)
	}
	return ret
}
func genWeekLinks(uid, val string) []*UnitStyle {
	ret := make([]*UnitStyle, 0)
	for i := 0; i < len(WEEK_LIST); i++ {
		wi := strconv.Itoa(i)
		var ul *UnitStyle
		if val == wi {
			ul = &UnitStyle{
				Name:  WEEK_LIST[i],
				Class: STYLE_CLASS_SEL,
			}
		} else {
			ul = &UnitStyle{
				Name: GenAnchorTagStr(WEEK_LIST[i], "/web/stats/"+uid+"/week/"+wi),
			}
		}
		ret = append(ret, ul)
	}
	return ret
}
func genDayLinks(uid, val string) []*UnitStyle {
	ret := make([]*UnitStyle, 0)
	sec := time.Seconds()
	for i := 0; i < 5; i++ {
		sec -= 86400
		t := time.SecondsToUTC(sec)
		d := fmt.Sprintf("%4d%2d%2d", t.Year, t.Month, t.Day)
		var ul *UnitStyle
		if val == d {
			ul = &UnitStyle{
				Name:  d,
				Class: STYLE_CLASS_SEL,
			}
		} else {
			ul = &UnitStyle{
				Name: GenAnchorTagStr(d, "/web/stats/"+uid+"/day/"+url.QueryEscape(d)),
			}
		}
		ret = append(ret, ul)
	}
	return ret
}
func genHourLinks(uid, val string) []*UnitStyle {
	ret := make([]*UnitStyle, 0)
	for i := 0; i < 24; i++ {
		h := strconv.Itoa(i)
		var ul *UnitStyle
		if val == h {
			ul = &UnitStyle{
				Name:  h,
				Class: STYLE_CLASS_SEL,
			}
		} else {
			ul = &UnitStyle{
				Name: GenAnchorTagStr(h, "/web/stats/"+uid+"/hour/"+h),
			}
		}
		ret = append(ret, ul)
	}
	return ret
}

func onInputError(ctx *web.Context) {
	ctx.Redirect(http.StatusFound, HOME_URL)
}

func onSystemError(ctx *web.Context) {
	ctx.Abort(http.StatusInternalServerError, ERR_MSG)
}

func setTmpCookie(ctx *web.Context, rt *oauth.RequestToken) {
	ctx.SetSecureCookie("tmp", rt.Token+","+rt.Secret, 3600)
}

func getTmpCookie(ctx *web.Context) *oauth.RequestToken {
	v, found := ctx.GetSecureCookie("tmp")
	if !found {
		return nil
	}
	rt := strings.Split(v, ",")
	if len(rt) != 2 {
		return nil
	}
	return &oauth.RequestToken{Token: rt[0], Secret: rt[1]}
}

func onLogin(ctx *web.Context) {
	rt, url, err := consumer.GetRequestTokenAndUrl(CALLBACK_URL)
	if err != nil {
		onSystemError(ctx)
		return
	}
	setTmpCookie(ctx, rt)
	ctx.Redirect(http.StatusFound, url)
}

func registUser(sess *mgo.Session, r io.Reader, at *oauth.AccessToken) *DigestwUser {
	var tu TwUser
	dec := json.NewDecoder(r)
	if err := dec.Decode(&tu); err != nil {
		return nil
	}
	du := NewDigestwUser(&tu, at)
	if _, err := du.Upsert(sess); err != nil {
		return nil
	}
	return du
}

func onCallback(ctx *web.Context) {
	rt := getTmpCookie(ctx)
	if rt == nil {
		onSystemError(ctx)
		return
	}
	oauth_verifier := ctx.Request.Params["oauth_verifier"]
	if oauth_verifier == "" {
		onSystemError(ctx)
		return
	}
	at, err := consumer.AuthorizeToken(rt, oauth_verifier)
	if err != nil {
		onSystemError(ctx)
		return
	}
	response, err := consumer.Get(
		"https://api.twitter.com/1/account/verify_credentials.json",
		map[string]string{"skip_status": "true"},
		at)
	if err != nil {
		onSystemError(ctx)
		return
	}
	defer response.Body.Close()
	sess := mgoPool.New()
	defer sess.Close()
	du := registUser(sess, response.Body, at)
	ctx.Redirect(http.StatusFound, "http://digestw.stoic.co.jp/web/stats/"+du.TwUser.Screen_Name+"/total/")
}

func onStatsDef(ctx *web.Context) {
	onStats(ctx, "mocchira", "total", "")
}

func onStats(ctx *web.Context, uid, unit, val string) {
	col, found := unit2col[unit]
	if !found {
		onInputError(ctx)
		return
	}
	var nsu StatsUnit
	var err os.Error
	sess := mgoPool.New()
	defer sess.Close()
	if err = nsu.Find(sess, col, uid, val); err != nil && err != mgo.NotFound {
		ctx.Logger.Println(err, col, uid, val)
		onSystemError(ctx)
		return
	}
	if err == mgo.NotFound {
		onInputError(ctx)
		return
	}
	fsort := func(kind, unit string, stats *Stats) {
		stats.GenOrderedKeys()
		keys := stats.Keys()
		ctx.Logger.Println(kind, unit, keys)
	}
	nsu.ForeachStats(fsort)
	if fmt, found := ctx.Params["fmt"]; found && fmt == "json" {
		if bytes, err := json.Marshal(&nsu); err != nil {
			ctx.Logger.Println(err, col, uid, val)
			onSystemError(ctx)
			return
		} else {
			ctx.Write(bytes)
			return
		}
	}
	bean := &Beans{
		Data:  &nsu,
		Units: make([]*UnitStyle, 0),
	}
	t := time.SecondsToUTC(time.Seconds() - 86400)
	unit2def["day"] = fmt.Sprintf("%4d%2d%2d", t.Year, t.Month, t.Day)
	for k, _ := range unit2col {
		if k == unit {
			if flink := unit2linkfun[unit]; flink != nil {
				bean.Links = flink(uid, val)
			}
			bean.Units = append(bean.Units, &UnitStyle{Name: k, Class: STYLE_CLASS_SEL})
		} else {
			anchor := `<a href="/web/stats/` +
				uid + "/" +
				k + "/" +
				url.QueryEscape(unit2def[k]) + `">` + k + `</a>`
			bean.Units = append(bean.Units, &UnitStyle{Name: anchor, Class: STYLE_CLASS_NON})
		}
	}
	if err := tplSet.Execute(ctx, "index.html", bean); err != nil {
		ctx.Logger.Println(err, col, uid, val)
		onSystemError(ctx)
		return
	}
	return
}

func main() {
	var consumerKey *string = flag.String("consumerkey", "RMA3YnQen7J0SDX67b5g", "")
	var consumerSecret *string = flag.String("consumersecret", "87GYFCqZz2k9VLcatBp7cpajzcdxRRPKfa3pMPtgW4", "")

	flag.Parse()

	// init
	consumer = oauth.NewConsumer(
		*consumerKey,
		*consumerSecret,
		oauth.ServiceProvider{
			RequestTokenUrl:   "https://api.twitter.com/oauth/request_token",
			AuthorizeTokenUrl: "https://api.twitter.com/oauth/authorize",
			AccessTokenUrl:    "https://api.twitter.com/oauth/access_token",
		})
	var err os.Error
	mgoPool, err = mgo.Mongo("localhost")
	if err != nil {
		panic(err)
	}
	defer mgoPool.Close()

	tplSet = new(template.Set)
	fmap := make(map[string]interface{})
	fmap["cut"] = CutString
	fmap["anchor"] = GenAnchorTagStr
	fmap["sn2url"] = GetTwitterAccountURL
	fmap["ht2url"] = GetSearchResultURL
	fmap["d2url"] = GetDomainURL
	tplSet.Funcs(fmap)
	tplSet = template.SetMust(
		tplSet.ParseTemplateFiles(
			"tpl/index.html",
			"tpl/place.html",
			"tpl/hashtag.html",
			"tpl/url.html",
			"tpl/domain.html",
			"tpl/sn.html",
			"tpl/tab.html",
			"tpl/link.html",
		))

	f, ferr := os.Create("server.log")
	if ferr != nil {
		panic(ferr)
	}
	logger := log.New(f, "", log.Ldate|log.Ltime)
	web.SetLogger(logger)
	web.Config.StaticDir = "../htdocs"
	web.Config.CookieSecret = "rep;oijgerke"

	web.Get("/web/login", onLogin)
	web.Get("/web/callback", onCallback)
	web.Get("/web/stats/([0-9a-zA-Z_]+)/([a-z]+)/([0-9 ]*)", onStats)
	web.Get("/web", onStatsDef)
	web.Run("127.0.0.1:8080")

}
