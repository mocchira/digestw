package main

import (
	"os"
	//"url"
	//"fmt"
	//"io"
	"log"
	"template"
	//"strconv"
	"json"
	"http"
	//"time"
	//"github.com/mrjones/oauth"
	web "github.com/hoisie/web.go"
	"launchpad.net/mgo"
)

const (
	HOME_URL = "http://192.168.56.101:8080/web/stats/mocchira/total/"
	ERR_MSG  = "Server Error"
)

var (
	mgoPool  *mgo.Session
	tplBase  *template.Set
	unit2col = map[string]string{
		"total": MGO_COL_STATS_TOTAL,
		"month": MGO_COL_STATS_MONTH,
		"week":  MGO_COL_STATS_WEEK,
		"day":   MGO_COL_STATS_DAY,
		"hour":  MGO_COL_STATS_HOUR,
	}
)

func onInputError(ctx *web.Context) {
	ctx.Redirect(http.StatusFound, HOME_URL)
}

func onSystemError(ctx *web.Context) {
	ctx.Abort(http.StatusInternalServerError, ERR_MSG)
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
	if fmt, found := ctx.Params["fmt"]; found && fmt == "json" {
		if bytes, err := json.Marshal(&nsu); err != nil {
			ctx.Logger.Println(err, col, uid, val)
			onSystemError(ctx)
		} else {
			return string(bytes)
		}
	}
	if err := tplBase.Execute(ctx, "index.html", &nsu); err != nil {
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

	tplBase = template.SetMust(template.ParseTemplateFiles("../htdocs/index.html", "../htdocs/table.html"))

	f, ferr := os.Create("server.log")
	if ferr != nil {
		panic(ferr)
	}
	logger := log.New(f, "", log.Ldate|log.Ltime)
	web.SetLogger(logger)
	web.Config.StaticDir = "../htdocs"

	web.Get("/web/stats/([0-9a-zA-Z_]+)/([a-z]+)/([0-9 ]*)", onStats)
	web.Run("0.0.0.0:8080")

}
