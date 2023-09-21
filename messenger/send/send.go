package send

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-resty/resty/v2"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

var (
	registered  = make(map[string]func(map[string]string) sender)
	msgCh       = make(chan *message, 10000)
	confCh      = make(chan struct{}, 1)
	name2sender = make(map[string]sender)
	rc          = resty.New()
)

type sender interface {
	send(*message) error
	getConf() map[string]string
}

type message struct {
	Sender  string   `json:"sender"`
	Title   string   `json:"title"`
	Content string   `json:"content"`
	MsgType string   `json:"msgtype"`
	Tos     []string `json:"tos"`
	Extra   string   `json:"extra"`

	ContentMap map[string]any
	ExtraMap   map[string]any
}

func init() {
	rc.RetryCount = 3
}

func Start() error {
	for {
		select {
		case <-confCh:
			handleConfig()
		case msg := <-msgCh:
		PRIORITY:
			for {
				select {
				case <-confCh:
					handleConfig()
				default:
					break PRIORITY
				}
			}
			handleMessage(msg)
		}
	}
}

func renderPretty(a any) string {
	bs, _ := json.MarshalIndent(a, "", " ")
	return string(bs)
}

func handleErr(info string, e error, resp *resty.Response, isOk func(dt map[string]any) bool) error {
	if e != nil {
		return e
	}

	dt := make(map[string]any)
	_ = json.Unmarshal(resp.Body(), &dt)
	if resp.StatusCode() != 200 || !isOk(dt) {
		return fmt.Errorf("%s httpcode=%v resp=%s", info, resp.StatusCode(), renderPretty(dt))
	}

	return nil
}

func PushConf() {
	confCh <- struct{}{}
}

func PushMessage(ctx *gin.Context) {
	m := &message{}
	if err := ctx.ShouldBindBodyWith(&m, binding.JSON); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}
	json.Unmarshal([]byte(cast.ToString(m.Content)), &m.ContentMap)
	json.Unmarshal([]byte(cast.ToString(m.Extra)), &m.ExtraMap)
	msgCh <- m
}

func readSenders() (confs []map[string]string, err error) {
	confs = make([]map[string]string, 0)
	err = viper.UnmarshalKey("senders", &confs)
	return
}

func handleConfig() {
	confs, err := readSenders()
	if err != nil {
		log.Println(err)
		return
	}
	for _, conf := range confs {
		name := conf["name"]
		if s, ok := name2sender[name]; !ok || !reflect.DeepEqual(conf, s.getConf()) {
			name2sender[name] = registered[conf["type"]](conf)
		}
	}
}

func handleMessage(msg *message) {
	s, ok := name2sender[msg.Sender]
	if !ok {
		log.Printf("cannot find sender with name %s\n", msg.Sender)
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Println(r)
			}
		}()
		if err := s.send(msg); err != nil {
			log.Printf("send failed message=%s\nerr=%v\n", renderPretty(msg), err)
		}
	}()
}
