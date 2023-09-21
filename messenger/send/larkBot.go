package send

import "github.com/samber/lo"

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
