package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"

	"messenger/global"
	"messenger/middleware"
	"messenger/send"
)

func main() {
	authConf, err := global.GetAuthConf()
	if err != nil {
		log.Fatalln(err)
	}
	appConf, err := global.GetAppConf()
	if err != nil {
		log.Fatalln(err)
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	v1 := r.Group("/v1").Use(middleware.Auth(authConf), middleware.Error2Resp())
	{
		v1.POST("/message", send.PushMessage)
		v1.POST("/uid/getbyphone", send.GetUIDByPhone)

		v1.POST("/senders", global.PushRemoteConf)
		v1.PUT("/senders", global.PushRemoteConf)
		v1.DELETE("/senders", global.PushRemoteConf)

	}

	eg := &errgroup.Group{}
	eg.Go(send.Start)
	eg.Go(func() error {
		return r.Run(fmt.Sprintf("%s:%s", appConf["ip"], appConf["port"]))
	})
	log.Println("start successfully...")
	if err := eg.Wait(); err != nil {
		log.Println(err)
	}
}
