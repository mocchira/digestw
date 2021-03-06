package main

import (
	"os"
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
	URL_CALLBACK    = "http://digestw.stoic.co.jp/web/callback"
	ERR_MSG_SYS     = "Server Error"
	ERR_MSG_NOT_GEN = "Requested page still not generated. Wait a while please."
	ERR_MSG_OAUTH   = "OAuth failed."
	ERR_MSG_COOKIE  = "Getting Temporary Cookie failed. Try again or confirm your browser's cookie setting on."
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
	User  *DigestwUser
	Data  *StatsUnit
	Units []*UnitStyle
	Links []*UnitStyle
	Error string
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
		t := time.SecondsToUTC(sec)
		d := fmt.Sprintf("%4d%2d%2d", t.Year, t.Month, t.Day)
		var ul *UnitStyle
		if val == d {
			ul = &UnitStyle{
				Name:  d,
				Class: STYLE_CLASS_SEL,
			}
		} else {
			urlPath := &url.URL{Path: d}
			ul = &UnitStyle{
				Name: GenAnchorTagStr(d, "/web/stats/"+uid+"/day/"+urlPath.EncodedPath()),
			}
		}
		ret = append(ret, ul)
		sec -= 86400
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

func genHomeURL(uid string) string {
	return "http://digestw.stoic.co.jp/web/stats/" + uid + "/total/"
}

func onInputError(ctx *web.Context, uid, errmsg string) {
	ctx.Redirect(http.StatusFound, genHomeURL(uid)+"?err="+url.QueryEscape(errmsg))
}

func onSystemError(ctx *web.Context) {
	ctx.Abort(http.StatusInternalServerError, ERR_MSG_SYS)
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
	rt, url, err := consumer.GetRequestTokenAndUrl(URL_CALLBACK)
	if err != nil {
		onSystemError(ctx)
		ctx.Logger.Println(err)
		return
	}
	setTmpCookie(ctx, rt)
	ctx.Redirect(http.StatusFound, url)
}

func onCallback(ctx *web.Context) {
	rt := getTmpCookie(ctx)
	if rt == nil {
		onInputError(ctx, "mocchira", ERR_MSG_COOKIE)
		ctx.Logger.Println("missing request token")
		return
	}
	oauth_verifier := ctx.Request.Params["oauth_verifier"]
	if oauth_verifier == "" {
		onInputError(ctx, "mocchira", ERR_MSG_OAUTH)
		ctx.Logger.Println("oauth verifier error")
		return
	}
	at, err := consumer.AuthorizeToken(rt, oauth_verifier)
	if err != nil {
		onInputError(ctx, "mocchira", ERR_MSG_OAUTH)
		ctx.Logger.Println(err)
		return
	}
	var tu TwUser
	if err := tu.GetFromAPI(consumer, at); err != nil {
		onInputError(ctx, "mocchira", ERR_MSG_OAUTH)
		ctx.Logger.Println(err)
		return
	}
	sess := mgoPool.New()
	defer sess.Close()
	if du, err := RegistUser(sess, &tu, at); err != nil {
		ctx.Logger.Println(err)
		onSystemError(ctx)
	} else {
		var tl TwTimeLine
		if err := tl.GetFromAPI(consumer, at, 200, du.SinceId); err != nil {
			onInputError(ctx, "mocchira", ERR_MSG_OAUTH)
			ctx.Logger.Println(err)
			return
		}
		done := make(chan int)
		go Crawl(sess, du, &tl, false, done)
		<-done
		ctx.Redirect(http.StatusFound, genHomeURL(du.TwUser.Screen_Name))
	}
}

func onStatsDef(ctx *web.Context) {
	onStats(ctx, "mocchira", "total", "")
}

func onStats(ctx *web.Context, uid, unit, val string) {
	col, found := unit2col[unit]
	if !found {
		onInputError(ctx, "mocchira", ERR_MSG_NOT_GEN)
		return
	}
	var nsu StatsUnit
	var du DigestwUser
	var err os.Error
	sess := mgoPool.New()
	defer sess.Close()
	if err = du.FindOne(sess, uid); err != nil && err != mgo.NotFound {
		ctx.Logger.Println(err, col, uid, val)
		onSystemError(ctx)
		return
	}
	if err == mgo.NotFound {
		onInputError(ctx, "mocchira", ERR_MSG_NOT_GEN)
		return
	}
	if err = nsu.Find(sess, col, uid, val); err != nil && err != mgo.NotFound {
		ctx.Logger.Println(err, col, uid, val)
		onSystemError(ctx)
		return
	}
	if err == mgo.NotFound {
		onInputError(ctx, uid, ERR_MSG_NOT_GEN)
		return
	}
	fsort := func(kind, unit string, stats *Stats) {
		stats.GenOrderedKeys()
		stats.Keys()
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
		User:  &du,
		Data:  &nsu,
		Units: make([]*UnitStyle, 0),
	}
	if msg, found := ctx.Params["err"]; found {
		bean.Error = msg
	}
	t := time.UTC()
	unit2def["hour"] = strconv.Itoa(t.Hour)
	unit2def["day"] = fmt.Sprintf("%4d%2d%2d", t.Year, t.Month, t.Day)
	unit2def["week"] = strconv.Itoa(t.Weekday)
	unit2def["month"] = strconv.Itoa(t.Month)
	for k, _ := range unit2col {
		if k == unit {
			if flink := unit2linkfun[unit]; flink != nil {
				bean.Links = flink(uid, val)
			}
			bean.Units = append(bean.Units, &UnitStyle{Name: k, Class: STYLE_CLASS_SEL})
		} else {
			urlPath := &url.URL{Path: unit2def[k]}
			href := "/web/stats/" + uid + "/" + k + "/" + urlPath.EncodedPath()
			anchor := GenAnchorTagStr(k, href)
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
	var consumerKey *string = flag.String("consumerkey", "xxx", "")
	var consumerSecret *string = flag.String("consumersecret", "xxx", "")
	var cookieSecret *string = flag.String("cookiesecret", "xxx", "")
	var mongoUrl *string = flag.String("mongourl", "xxx", "")

	flag.Parse()

	consumer = oauth.NewConsumer(
		*consumerKey,
		*consumerSecret,
		oauth.ServiceProvider{
			RequestTokenUrl:   "https://api.twitter.com/oauth/request_token",
			AuthorizeTokenUrl: "https://api.twitter.com/oauth/authorize",
			AccessTokenUrl:    "https://api.twitter.com/oauth/access_token",
		})
	var err os.Error
	mgoPool, err = mgo.Mongo(*mongoUrl)
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
	web.Config.CookieSecret = *cookieSecret

	web.Get("/web/login", onLogin)
	web.Get("/web/callback", onCallback)
	web.Get("/web/stats/([0-9a-zA-Z_]+)/([a-z]+)/([0-9 ]*)", onStats)
	web.Get("/web", onStatsDef)
	web.Run("127.0.0.1:8080")

}
