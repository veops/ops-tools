package send

func init() {
	registered["wechatBot"] = func(conf map[string]string) sender {
		return &wechatBot{conf: conf}
	}
}

type wechatBot struct {
	conf map[string]string
}

// send wechat bot message
//
//	https://developer.work.weixin.qq.com/document/path/99110
func (w *wechatBot) send(msg *message) error {
	resp, err := rc.R().
		SetBody(map[string]any{
			"msgtype":   msg.MsgType,
			msg.MsgType: msg.Content,
		}).
		Post(w.conf["url"])

	return handleErr("wechat bot send failed", err, resp, func(dt map[string]any) bool { return dt["errcode"] == 0.0 })
}

func (w *wechatBot) getConf() map[string]string {
	return w.conf
}
