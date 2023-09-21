package send

import (
	"crypto/tls"
	"log"
	"sync"

	"github.com/samber/lo"
	"github.com/spf13/cast"
	"gopkg.in/gomail.v2"
)

func init() {
	registered["email"] = func(conf map[string]string) sender {
		return &email{conf: conf}
	}
}

type email struct {
	conf map[string]string
	once sync.Once
	d    *gomail.Dialer
}

func (e *email) send(msg *message) (err error) {
	e.once.Do(func() {
		e.d = lo.TernaryF(e.conf["password"] == "",
			func() *gomail.Dialer {
				return &gomail.Dialer{Host: e.conf["host"], Port: cast.ToInt(e.conf["port"])}
			}, func() *gomail.Dialer {
				return gomail.NewDialer(e.conf["host"], cast.ToInt(e.conf["port"]), e.conf["account"], e.conf["password"])
			})
		e.d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	})

	m := gomail.NewMessage()
	m.SetHeader("From", e.conf["account"])
	m.SetHeader("To", msg.Tos...)
	m.SetHeader("Subject", msg.Title)
	m.SetHeader("Cc", msg.Ccs...)
	m.SetBody(msg.MsgType, msg.Content)

	if err = e.d.DialAndSend(m); err != nil {
		log.Println(err)
	}

	return
}

func (e *email) getConf() map[string]string {
	return e.conf
}
