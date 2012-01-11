package main

import (
	"flag"
	"fmt"
	oauth "github.com/alloy-d/goauth"
	"launchpad.net/mgo"
	"log"
	"os"
	"time"
)

const (
	MODE_TEST      = "test"
	MODE_REG_USER  = "user"
	MODE_OAUTH_OOB = "oauth"
	MODE_DEFAULT   = "default"
)

func main() {
	var consumerKey *string = flag.String("consumerkey", "xxx", "")
	var consumerSecret *string = flag.String("consumersecret", "xxx", "")
	var mongoUrl *string = flag.String("mongourl", "xxx", "")
	var count *int = flag.Int("count", 100, "")
	var mode *string = flag.String("mode", "default", "")
	uid := flag.Int64("uid", 0, "")
	screen_name := flag.String("screen_name", "xxx", "")
	profile_image_url := flag.String("profile_image_url", "xxx", "")
	utc_offset := flag.Int64("utc_offset", 0, "")
	token := flag.String("token", "xxx", "")
	secret := flag.String("secret", "xxx", "")

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
	c := &oauth.OAuth{
		ConsumerKey:     *consumerKey,
		ConsumerSecret:  *consumerSecret,
		RequestTokenURL: "https://api.twitter.com/oauth/request_token",
		OwnerAuthURL:    "https://api.twitter.com/oauth/authorize",
		AccessTokenURL:  "https://api.twitter.com/oauth/access_token",
		Callback:        "oob",
		SignatureMethod: oauth.HMAC_SHA1,
	}
	mgPool, err := mgo.Mongo(*mongoUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer mgPool.Close()

	switch *mode {
	case MODE_REG_USER:
		tu := &TwUser{
			Id:                *uid,
			Screen_Name:       *screen_name,
			Profile_Image_Url: profile_image_url,
			UTC_Offset:        utc_offset,
		}
		if du, err := RegistUser(mgPool, tu, *token, *secret); err != nil {
			log.Fatal(err)
		} else {
			fmt.Println("id:" + du.TwUser.Screen_Name)
		}
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
	case MODE_OAUTH_OOB:
		err = c.GetRequestToken()
		if err != nil {
			log.Fatal(err)
		}
		url, err := c.AuthorizationURL()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("(1) Go to: " + url)
		fmt.Println("(2) Grant access, you should get back a verification code.")
		fmt.Println("(3) Enter that verification code here: ")
		verificationCode := ""
		fmt.Scanln(&verificationCode)
		err = c.GetAccessToken(verificationCode)
		if err != nil {
			log.Fatal(err)
		}
		var tu TwUser
		if err := tu.GetFromAPI(c); err != nil {
			log.Fatal(err)
		}
		if du, err := RegistUser(mgPool, &tu, c.AccessToken(), c.AccessSecret()); err != nil {
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
			iter := dulist[0].Find(mgPool, time.Now().Unix())
			for iter.Next(&dulist[idx]) {
				var tl TwTimeLine
				c.SetAccessToken(dulist[idx].Token)
				c.SetAccessSecret(dulist[idx].Secret)
				if err := tl.GetFromAPI(c, *count, dulist[idx].SinceId); err != nil {
					log.Println(err)
					continue
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
