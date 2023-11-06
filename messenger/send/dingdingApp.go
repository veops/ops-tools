package send

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/samber/lo"
	"github.com/spf13/cast"
)

const (
	dingdingTokenURL  = "https://api.dingtalk.com/v1.0/oauth2/accessToken"
	dingdingSendURL   = "https://api.dingtalk.com/v1.0/robot/oToMessages/batchSend"
	dingdingGetUIDURL = "https://oapi.dingtalk.com/topapi/v2/user/getbymobile"
)

func init() {
	registered["dingdingApp"] = func(conf map[string]string) sender {
		return &dingdingApp{conf: conf}
	}
}

type dingdingApp struct {
	conf          map[string]string
	mtx           sync.Mutex
	token         string
	tokenExpireAt time.Time
}

// send dingtalk app message
//
//	https://open.dingtalk.com/document/orgapp/chatbots-send-one-on-one-chat-messages-in-batches
func (d *dingdingApp) send(msg *message) (err error) {
	if err = d.checkToken(); err != nil {
		return
	}

	if msg.Simple {
		switch msg.MsgType {
		case simpleText:
			msg.MsgType = "sampleText"
			msg.ContentMap = map[string]any{
				"content": msg.Content,
			}
		case simpleMarkdown:
			msg.MsgType = "sampleMarkdown"
			msg.ContentMap = map[string]any{
				"title": msg.Title,
				"text":  msg.Content,
			}
		default:
			return fmt.Errorf("sender type %s does not support simple type %s", d.conf["type"], msg.MsgType)
		}
	}
	bs, _ := json.Marshal(msg.ContentMap)
	resp, err := rc.R().
		SetHeader("x-acs-dingtalk-access-token", d.token).
		SetBody(lo.Assign(msg.ExtraMap, map[string]any{
			"robotCode": d.conf["robotCode"],
			"userIds":   msg.Tos,
			"msgKey":    msg.MsgType,
			"msgParam":  string(bs),
		})).
		Post(dingdingSendURL)

	return handleErr("send to dingding app failed", err, resp, func(dt map[string]any) bool {
		_, ok := dt["processQueryKey"]
		return ok
	})
}

func (d *dingdingApp) getConf() map[string]string {
	return d.conf
}

// getUIDByPhone
//
//	https://open.dingtalk.com/document/orgapp/query-users-by-phone-number
func (d *dingdingApp) getUIDByPhone(phone string) (uid string, err error) {
	if err = d.checkToken(); err != nil {
		return
	}

	type res struct {
		Result struct {
			ExclusiveAccountUseridList []string `json:"exclusive_account_userid_list"`
			Userid                     string   `json:"userid"`
		} `json:"result"`
	}
	r := &res{}

	resp, err := rc.R().
		SetQueryParam("access_token", d.token).
		SetBody(map[string]any{
			"support_exclusive_account_search": true,
			"mobile":                           phone,
		}).
		SetResult(&r).
		Post(dingdingGetUIDURL)

	if err = handleErr("get uid by phone with dingding app failed", err, resp, func(dt map[string]any) bool { return dt["errcode"] == 0.0 }); err != nil {
		return
	}

	uid = r.Result.Userid

	return
}

func (d *dingdingApp) checkToken() (err error) {
	now := time.Now()
	if !(d.token == "" || d.tokenExpireAt.Before(now)) {
		return
	}

	d.mtx.Lock()
	defer d.mtx.Unlock()
	if d.token == "" || d.tokenExpireAt.Before(now) {
		var resp *resty.Response
		resp, err = rc.R().
			SetBody(map[string]string{
				"appKey":    d.conf["appKey"],
				"appSecret": d.conf["appSecret"],
			}).
			Post(dingdingTokenURL)

		// errcode 0 doesnot return as doc when successful
		// https://open.dingtalk.com/document/orgapp/obtain-orgapp-token?spm=ding_open_doc.document.0.0.454d4a97mHIEGp
		if err = handleErr("get dingding access token failed", err, resp, func(dt map[string]any) bool {
			v, ok := dt["errcode"]
			return !ok || v == 0.0
		}); err != nil {
			return
		}

		dt := make(map[string]any)
		_ = json.Unmarshal(resp.Body(), &dt)
		d.token = cast.ToString(dt["accessToken"])
		d.tokenExpireAt = now.Add(time.Second * time.Duration(cast.ToInt((dt["expireIn"]))))
	}

	return
}
