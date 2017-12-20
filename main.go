package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	mgo "github.com/globalsign/mgo"
	goflags "github.com/jessevdk/go-flags"
)

type Flags struct {
	MongoURL string `long:"mongo" default:""`
}

var flags Flags

func init() {
	mgo.SetDebug(true)
	mgo.SetLogger(log.New(os.Stderr, "[MONGODB] ", log.LstdFlags|log.Llongfile))
}

func main() {
	parser := goflags.NewParser(&flags, goflags.HelpFlag|goflags.PassDoubleDash|goflags.IgnoreUnknown)
	if restedArgs, err := parser.ParseArgs(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	} else {
		os.Args = append(os.Args[:1], restedArgs...)
	}

	const sessionCount = 16
	sessions := make([]*mgo.Session, sessionCount)
	if session, err := mgo.Dial(flags.MongoURL); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	} else {
		sessions[0] = session
		for i := 1; i < sessionCount; i++ {
			sessions[i] = sessions[0].Copy()
		}
	}

	var sessionIndex int32 = 0

	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		for i, session := range sessions {
			if err := session.Ping(); err != nil {
				c.String(http.StatusInternalServerError, "Ping Failed on %d: %s", i, err)
				return
			}
		}
		c.String(http.StatusOK, "Success")
	})
	r.GET("/test", func(c *gin.Context) {
		index := atomic.AddInt32(&sessionIndex, 1)
		if _, err := sessions[index%sessionCount].DB("test").C("test").Find(nil).Count(); err != nil {
			c.String(http.StatusInternalServerError, "Test Failed on %d: %s", index, err)
		} else {
			c.String(http.StatusOK, "Test Success on %d", index)
		}
	})
	r.Run()
}
