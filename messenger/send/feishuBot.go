package send

import (
	"fmt"

	"github.com/samber/lo"
)

func init() {
	registered["feishuBot"] = func(conf map[string]string) sender {
		return &feishuBot{conf: conf}
	}
}

type feishuBot struct {
	conf map[string]string
}

// send feishu bot message
//
//	https://open.feishu.cn/document/client-docs/bot-v3/add-custom-bot
func (f *feishuBot) send(msg *message) error {
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
		SetBody(lo.Assign(msg.ExtraMap, map[string]any{
			"msg_type": msg.MsgType,
			"content":  msg.ContentMap,
		})).
		Post(f.conf["url"])

	return handleErr("send to feishu bot failed", err, resp, func(dt map[string]any) bool { return dt["code"] == 0.0 })
}

func (f *feishuBot) getConf() map[string]string {
	return f.conf
}
