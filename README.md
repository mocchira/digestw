Digestw - Twitter's Timeline Digest
===================================

A simple web application powerd by golang and mongodb and twitter's API.
I am a new golanger and this is my first go project.
The main purpose of this project is to understand and exercise and evaluate golang.
So there may be irrelevant or not idiomatic codes,
However I hope this project helps beginners learn golang and give a chance to have an interest in golang!

Dependencies
------------

* golang(http://golang.org/doc/install.html)
* mongodb(http://www.mongodb.org/downloads)

Patch
-----
While developing this project, I discovered some bugs of go http package.
Since those bugs can cause dead locked my go programs, To keep my programs stable on production env
need to apply the following patches.

* http.Client can get into dead locked(http://code.google.com/p/go/issues/detail?id=2616)
* http.Request.write doesn't handle bw.Flush()'s error(http://code.google.com/p/go/issues/detail?id=2645)

Dependencies of goinstallable packages
-------------------------------------

* oauth(https://github.com/mrjones/oauth) -- `goinstall github.com/mrjones/oauth`
* web.go(https://github.com/hoisie/web.go) -- `goinstall github.com/hoisie/web.go`
* oauth(https://launchpad.net/mgo) -- `goinstall launchpad.net/mgo`
* oauth(https://launchpad.net/gobson/bson) -- `goinstall launchpad.net/gobson/bson`

Install
-------
    
    cd batch && make
    cd web && make

Run
---

### batch
need to execute at fixed intervals. (ex. 5min by cron)
    
    ./digestw_crawler -consumerkey "yourconsumerkey" -consumersecret "yourconsumersecret" >> digestw.log 2>&1

### web
run on 8080 port
    
    ./digestw_web -consumerkey "yourconsumerkey" -consumersecret "yourconsumersecret" -cookiesecret "yourcookiese" -mongourl "yourhost" &

Site
----
http://digestw.stoic.co.jp/web

About
-----
digestw was written by [Yoshiyuki Kanno](http://www.twitter.com/mocchira)

