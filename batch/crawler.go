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

func main() {
	var consumerKey *string = flag.String("consumerkey", "xxx", "")
	var consumerSecret *string = flag.String("consumersecret", "xxx", "")
	var mongoUrl *string = flag.String("mongourl", "xxx", "")
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
	mgPool, err := mgo.Mongo(*mongoUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer mgPool.Close()

	switch *mode {
	case MODE_TEST:
		var tl TwTimeLine
		if err := tl.Get(os.Stdin); err != nil {
			log.Fatal(err)
		}
		var du DigestwUser
		du.TwUser.Screen_Name = "mocchira"
		done := make(chan int)
		go Crawl(mgPool, &du, &tl, true, done)
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
		var tu TwUser
		if err := tu.GetFromAPI(c, accessToken); err != nil {
			log.Fatal(err)
		}
		if du, err := RegistUser(mgPool, &tu, accessToken); err != nil {
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
				var tl TwTimeLine
				if err := tl.GetFromAPI(c, &dulist[idx].AccessToken, *count, dulist[idx].SinceId); err != nil {
					log.Println(err)
				}
				go Crawl(mgPool, &dulist[idx], &tl, true, done)
				log.Printf("[go]idx:%d sn:%s", idx, dulist[idx].TwUser.Screen_Name)
				idx++
			}
			for ; idx > 0; idx-- {
				<-done
				log.Printf("[go]idx:%d done", idx-1)
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
