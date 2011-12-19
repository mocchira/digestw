package main

import (
	"os"
	"log"
	"fmt"
	"template"
	"json"
	"http"
	"url"
	"time"
	web "github.com/hoisie/web.go"
	"launchpad.net/mgo"
)

const (
	HOME_URL        = "http://192.168.56.101:8080/web/stats/mocchira/total/"
	ERR_MSG         = "Server Error"
	STYLE_CLASS_NON = "non"
	STYLE_CLASS_SEL = "selected"
)

var (
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
)

type UnitStyle struct {
	Class string
	Name  string
}
type Beans struct {
	Data  *StatsUnit
	Units []*UnitStyle
}

func onInputError(ctx *web.Context) {
	ctx.Redirect(http.StatusFound, HOME_URL)
}

func onSystemError(ctx *web.Context) {
	ctx.Abort(http.StatusInternalServerError, ERR_MSG)
}

func onStatsDef(ctx *web.Context) string {
	return onStats(ctx, "mocchira", "total", "")
}

func onStats(ctx *web.Context, uid, unit, val string) string {
	col, found := unit2col[unit]
	if !found {
		onInputError(ctx)
		return ""
	}
	var nsu StatsUnit
	var err os.Error
	sess := mgoPool.New()
	defer sess.Close()
	if err = nsu.Find(sess, col, uid, val); err != nil && err != mgo.NotFound {
		ctx.Logger.Println(err, col, uid, val)
		onSystemError(ctx)
		return ""
	}
	if err == mgo.NotFound {
		onInputError(ctx)
		return ""
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
		} else {
			return string(bytes)
		}
	}
	bean := &Beans{
		Data:  &nsu,
		Units: make([]*UnitStyle, 0),
	}
	t := time.UTC()
	unit2def["day"] = fmt.Sprintf("%4d%2d%2d", t.Year, t.Month, t.Day)
	for k, _ := range unit2col {
		if k == unit {
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
		return ""
	}
	return ""
}

func main() {
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
		))

	f, ferr := os.Create("server.log")
	if ferr != nil {
		panic(ferr)
	}
	logger := log.New(f, "", log.Ldate|log.Ltime)
	web.SetLogger(logger)
	web.Config.StaticDir = "../htdocs"

	web.Get("/", onStatsDef)
	web.Get("/web/stats/([0-9a-zA-Z_]+)/([a-z]+)/([0-9 ]*)", onStats)
	web.Run("0.0.0.0:8080")

}
