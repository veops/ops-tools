package send

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/samber/lo"
	"github.com/spf13/cast"
)

const (
	wechatTokenURL = "https://qyapi.weixin.qq.com/cgi-bin/gettoken"
	wechatSendURL  = "https://qyapi.weixin.qq.com/cgi-bin/message/send"
)

func init() {
	registered["wechatApp"] = func(conf map[string]string) sender {
		return &wechatApp{conf: conf}
	}
}

type wechatApp struct {
	conf          map[string]string
	mtx           sync.Mutex
	token         string
	tokenExpireAt time.Time
}

// send wechat app message
//
//	https://developer.work.weixin.qq.com/document/path/90236
func (w *wechatApp) send(msg *message) (err error) {
	if err = w.checkToken(); err != nil {
		return
	}
	resp, err := rc.R().
		SetQueryParam("access_token", w.token).
		SetBody(lo.Assign(msg.ExtraMap, map[string]any{
			"touser":    strings.Join(msg.Tos, "|"),
			"agentid":   w.conf["agentid"],
			"msgtype":   msg.MsgType,
			msg.MsgType: msg.ContentMap,
		})).
		Post(wechatSendURL)

	return handleErr("send to wechat app failed", err, resp, func(dt map[string]any) bool { return dt["errcode"] == 0.0 })
}

func (w *wechatApp) getConf() map[string]string {
	return w.conf
}

func (w *wechatApp) checkToken() (err error) {
	now := time.Now()
	if !(w.token == "" || w.tokenExpireAt.Before(now)) {
		return nil
	}

	w.mtx.Lock()
	defer w.mtx.Unlock()
	if w.token == "" || w.tokenExpireAt.Before(now) {
		var resp *resty.Response
		resp, err = rc.R().
			SetQueryParams(map[string]string{
				"corpid":     w.conf["corpid"],
				"corpsecret": w.conf["corpsecret"],
			}).
			Get(wechatTokenURL)

		if err = handleErr("wechat get access token failed", err, resp, func(dt map[string]any) bool { return dt["errcode"] == 0.0 }); err != nil {
			return
		}

		dt := make(map[string]any)
		_ = json.Unmarshal(resp.Body(), &dt)
		w.token = cast.ToString(dt["access_token"])
		w.tokenExpireAt = now.Add(time.Second * time.Duration(cast.ToInt((dt["expires_in"]))))
	}

	return
}
