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
	feishuTokenURL  = "https://open.feishu.cn/open-apis/auth/v3/app_access_token/internal"
	feishuSendURL   = "https://open.feishu.cn/open-apis/message/v4/batch_send/"
	feishuGetUIDURL = "https://open.feishu.cn/open-apis/contact/v3/users/batch_get_id"
)

func init() {
	registered["feishuApp"] = func(conf map[string]string) sender {
		return &feishuApp{conf: conf}
	}
}

type feishuApp struct {
	conf          map[string]string
	mtx           sync.Mutex
	token         string
	tokenExpireAt time.Time
}

// send feishu app message
//
//	https://open.feishu.cn/document/server-docs/im-v1/batch_message/send-messages-in-batches
func (f *feishuApp) send(msg *message) (err error) {
	if err = f.checkToken(); err != nil {
		return
	}

	if msg.Simple {
		switch msg.MsgType {
		case simpleText:
			msg.ContentMap = map[string]any{
				msg.MsgType: msg.Content,
			}
		default:
			return fmt.Errorf("sender type %s does not support simple type %s", f.conf["type"], msg.MsgType)
		}
	}
	resp, err := rc.R().
		SetAuthToken(f.token).
		SetQueryParam("receive_id_type", "user_id").
		SetBody(lo.Assign(msg.ExtraMap, map[string]any{
			"user_ids": msg.Tos,
			"msg_type": msg.MsgType,
			"content":  msg.ContentMap,
		})).
		Post(feishuSendURL)

	return handleErr("send to feishu app failed", err, resp, func(dt map[string]any) bool { return dt["code"] == 0.0 })
}

func (f *feishuApp) getConf() map[string]string {
	return f.conf
}

// getUIDByPhone
//
//	https://open.feishu.cn/open-apis/contact/v3/users/batch_get_id
func (f *feishuApp) getUIDByPhone(phone string) (uid string, err error) {
	if err = f.checkToken(); err != nil {
		return
	}

	type res struct {
		Data struct {
			UserList []struct {
				UserID string `json:"user_id"`
			} `json:"user_list"`
		} `json:"data"`
	}
	r := &res{}

	resp, err := rc.R().
		SetAuthToken(f.token).
		SetQueryParam("user_id_type", "user_id").
		SetBody(map[string]any{
			"mobiles": []string{phone},
		}).
		SetResult(r).
		Post(feishuGetUIDURL)

	if err = handleErr("get uid by phone with feishu app failed", err, resp, func(dt map[string]any) bool { return dt["code"] == 0.0 }); err != nil {
		return
	}

	if len(r.Data.UserList) > 0 {
		uid = r.Data.UserList[0].UserID
	}

	return
}

func (f *feishuApp) checkToken() (err error) {
	now := time.Now()
	if !(f.token == "" || f.tokenExpireAt.Before(now)) {
		return
	}

	f.mtx.Lock()
	defer f.mtx.Unlock()
	if f.token == "" || f.tokenExpireAt.Before(now) {
		var resp *resty.Response
		resp, err = rc.R().
			SetBody(map[string]string{"app_id": f.conf["app_id"], "app_secret": f.conf["app_secret"]}).
			Post(feishuTokenURL)

		if err = handleErr("get feishu token failed", err, resp, func(dt map[string]any) bool { return dt["code"] == 0.0 }); err != nil {
			return
		}

		dt := make(map[string]any)
		_ = json.Unmarshal(resp.Body(), &dt)
		f.token = cast.ToString(dt["app_access_token"])
		f.tokenExpireAt = now.Add(time.Second * time.Duration(cast.ToInt((dt["expire"]))))
	}

	return
}
