package main

import (
	"os"
	"flag"
	"fmt"
	"log"
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
	mgPool *mgo.Session
)

func main() {
	var consumerKey *string = flag.String("consumerkey", "xxx", "")
	var consumerSecret *string = flag.String("consumersecret", "xxx", "")
	var count *int = flag.Int("count", 100, "")
	var mode *string = flag.String("mode", "default", "")

	flag.Parse()

	// stop dual executing
	dl := NewDirProcessLocker("lock")
	if err := dl.Lock(); err != nil {
		if err == ProcessExist {
			return
		}
		log.Fatal(err)
	}
	defer dl.Unlock()

	// init
	c := oauth.NewConsumer(
		*consumerKey,
		*consumerSecret,
		oauth.ServiceProvider{
			RequestTokenUrl:   "https://api.twitter.com/oauth/request_token",
			AuthorizeTokenUrl: "https://api.twitter.com/oauth/authorize",
			AccessTokenUrl:    "https://api.twitter.com/oauth/access_token",
		})
	//c.Debug(true)
	var err os.Error
	mgPool, err = mgo.Mongo("localhost")
	if err != nil {
		log.Fatal(err)
	}
	defer mgPool.Close()

	switch *mode {
	case MODE_TEST:
		// js test
		var du DigestwUser
		du.TwUser.Screen_Name = "mocchira"
		done := make(chan int)
		go Crawl(mgPool, c, &du, os.Stdin, *count, true, done)
		<-done
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
		if du, err := RegistUser(mgPool, response.Body, accessToken); err != nil {
			log.Fatal(err)
		} else {
			fmt.Println("id:" + du.TwUser.Screen_Name)
		}
	default:
		var idx int
		dulist := [CRAWL_UNIT]DigestwUser{}
		done := make(chan int)
		for true {
			idx = 0
			iter := dulist[0].Find(mgPool, time.Seconds())
			for iter.Next(&dulist[idx]) {
				go Crawl(mgPool, c, &dulist[idx], nil, *count, true, done)
				log.Printf("[go]idx:%d sn:%s", idx, dulist[idx].TwUser.Screen_Name)
				idx++
			}
			for ; idx > 0; idx-- {
				<-done
				log.Printf("[go]idx:%d done", idx)
			}
			if err := iter.Err(); err != nil {
				log.Fatal(err)
			}
			if idx < CRAWL_UNIT {
				break
			}
		}
		return
	}

}
