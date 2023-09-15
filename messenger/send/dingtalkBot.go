package send

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"time"
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
	r := rc.R().
		SetBody(map[string]any{
			"msgtype":   msg.MsgType,
			msg.MsgType: msg.Content,
		})
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
