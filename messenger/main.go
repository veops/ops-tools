package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"golang.org/x/sync/errgroup"

	"messenger/middleware"
	"messenger/send"
)

var (
	k = koanf.New(".")
)

func main() {
	f := file.Provider("conf.yaml")
	if err := k.Load(f, yaml.Parser()); err != nil {
		log.Fatalln(err)
	}
	f.Watch(func(event interface{}, err error) {
		if err != nil {
			log.Fatalln(err)
			return
		}
		k.Load(f, yaml.Parser())
		confs := make(map[string][]map[string]string)
		k.Unmarshal("senders", &confs)
		send.PushConf(confs)
	})
	confs := make(map[string][]map[string]string)
	k.Unmarshal("senders", &confs)
	send.PushConf(confs)

	authConfs := make([]map[string]string, 0)
	if err := k.Unmarshal("auths", &authConfs); err != nil {
		log.Fatalln(err)
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.MaxMultipartMemory = 100 << 20 //100MB
	v1 := r.Group("/v1").Use(middleware.Auth(authConfs))
	{
		v1.POST("/message", send.PushMessage)
	}

	eg := &errgroup.Group{}
	eg.Go(send.Start)
	eg.Go(func() error {
		return r.Run(fmt.Sprintf("%s:%d", k.String("app.ip"), k.Int("app.port")))
	})
	log.Println("start successfully...")
	if err := eg.Wait(); err != nil {
		log.Println(err)
	}
}
