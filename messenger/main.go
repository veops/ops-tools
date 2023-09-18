package main

import (
	"fmt"
	"log"

	"github.com/fsnotify/fsnotify"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"

	"messenger/middleware"
	"messenger/send"
)

func main() {
	viper.SetConfigName("conf")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.WatchConfig()
	viper.OnConfigChange(func(in fsnotify.Event) {
		if in.Has(fsnotify.Write) {
			send.PushConf()
		}
	})
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalln(err)
	}
	send.PushConf()

	authConfs := make([]map[string]string, 0)
	if err := viper.UnmarshalKey("auths", &authConfs); err != nil {
		log.Fatalln(err)
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	v1 := r.Group("/v1").Use(middleware.Auth(authConfs))
	{
		v1.POST("/message", send.PushMessage)
	}

	eg := &errgroup.Group{}
	eg.Go(send.Start)
	eg.Go(func() error {
		return r.Run(fmt.Sprintf("%s:%d", viper.GetString("app.ip"), viper.GetInt("app.port")))
	})
	if err := eg.Wait(); err != nil {
		log.Println(err)
	}
}
