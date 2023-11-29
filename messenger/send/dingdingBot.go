package send

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/samber/lo"
)

func init() {
	registered["dingdingBot"] = func(conf map[string]string) sender {
		return &dingdingBot{conf: conf}
	}
}

type dingdingBot struct {
	conf map[string]string
}

// send dingtal bot message
//
//	https://open.dingtalk.com/document/orgapp/custom-bot-creation-and-installation
func (d *dingdingBot) send(msg *message) error {
	if msg.Simple {
		switch msg.MsgType {
		case simpleText:
			msg.ContentMap = map[string]any{
				"content": msg.Content,
			}
		case simpleMarkdown:
			msg.ContentMap = map[string]any{
				"text": msg.Content,
			}
		default:
			return fmt.Errorf("sender type %s does not support simple type %s", d.conf["type"], msg.MsgType)
		}
	}
	if msg.MsgType == simpleMarkdown {
		msg.ContentMap["title"] = msg.Title
	}

	at := make(map[string]any)
	if len(msg.Ats) > 0 {
		switch msg.MsgType {
		case simpleText, simpleMarkdown:
			at["isAtAll"] = lo.Contains(msg.Ats, "@all")
			at["atUserIds"] = lo.Without(msg.Ats, "@all")
			msg.ContentMap["text"] = fmt.Sprintf("%v \n %s",
				msg.ContentMap["text"],
				strings.Join(lo.Map(msg.Ats, func(s string, _ int) string { return fmt.Sprintf("@%s", s) }), " "))
		}
	}

	r := rc.R().
		SetBody(lo.Assign(msg.ExtraMap, map[string]any{
			"msgtype":   msg.MsgType,
			msg.MsgType: msg.ContentMap,
			"at":        at,
		}))
	if d.conf["token"] != "" {
		ts := time.Now().UnixMilli()
		sts := fmt.Sprintf("%d\n%s", ts, d.conf["token"])
		mac := hmac.New(sha256.New, []byte(d.conf["token"]))
		_, _ = mac.Write([]byte(sts))
		sign := url.QueryEscape(base64.StdEncoding.EncodeToString(mac.Sum(nil)))
		r.SetQueryParams(map[string]string{
			"timestamp": fmt.Sprintf("%d", ts),
			"sign":      sign,
		})
	}
	resp, err := r.Post(d.conf["url"])

	return handleErr("send to dingding bot failed", err, resp, func(dt map[string]any) bool { return dt["errcode"] == 0.0 })
}

func (d *dingdingBot) getConf() map[string]string {
	return d.conf
}
