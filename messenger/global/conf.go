package global

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/samber/lo"
)

var (
	k   = koanf.New(".")
	cbs = make([]func(), 0)
	mtx = &sync.RWMutex{}
	p   = yaml.Parser()
)

func init() {
	f := file.Provider("conf/conf.yaml")
	if err := k.Load(f, p); err != nil {
		log.Fatalln(err)
	}
	f.Watch(func(event interface{}, err error) {
		if err != nil {
			log.Fatalln(err)
			return
		}
		k.Load(f, p)
		doCallbacks()
	})
}

func doCallbacks() {
	mtx.RLock()
	defer mtx.RUnlock()

	for _, cb := range cbs {
		cb()
	}
}

func RegisterWatchCallbacks(fs ...func()) {
	for _, f := range fs {
		f()
	}
	cbs = append(cbs, fs...)
}

func GetAppConf() (conf map[string]string, err error) {
	mtx.RLock()
	defer mtx.RUnlock()

	conf = make(map[string]string)
	err = k.Unmarshal("app", &conf)

	return
}

func GetAuthConf() (confs []map[string]string, err error) {
	mtx.RLock()
	defer mtx.RUnlock()

	confs = make([]map[string]string, 0)
	err = k.Unmarshal("auths", &confs)

	return
}

func GetSenders() (senders []map[string]string, err error) {
	mtx.RLock()
	defer mtx.RUnlock()

	typedSenders := make(map[string][]map[string]string, 0)
	err = k.Unmarshal("senders", &typedSenders)

	senders = make([]map[string]string, 0)
	for t, ss := range typedSenders {
		for _, s := range ss {
			s["type"] = t
			senders = append(senders, s)
		}
	}

	return
}

func PushRemoteConf(ctx *gin.Context) {
	update := make(map[string][]map[string]string)
	if err := ctx.ShouldBindBodyWith(&update, binding.JSON); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	mtx.Lock()
	defer mtx.Unlock()

	pre, cur := make(map[string][]map[string]string), make(map[string][]map[string]string)
	k.Unmarshal("senders", &pre)
	k.Unmarshal("senders", &cur)
	switch ctx.Request.Method {
	case "POST":
		for t, ss := range update {
			cur[t] = ss
		}
	case "PUT":
		for t, ss := range update {
			m := lo.Assign(
				lo.SliceToMap(cur[t], func(s map[string]string) (string, map[string]string) { return s["name"], s }),
				lo.SliceToMap(ss, func(s map[string]string) (string, map[string]string) { return s["name"], s }),
			)
			cur[t] = lo.MapToSlice(m, func(_ string, v map[string]string) map[string]string { return v })
		}
	case "DELETE":
		for t, ss := range update {
			m := lo.OmitByKeys(
				lo.SliceToMap(cur[t], func(s map[string]string) (string, map[string]string) { return s["name"], s }),
				lo.Map(ss, func(v map[string]string, _ int) string { return v["name"] }),
			)
			cur[t] = lo.MapToSlice(m, func(_ string, v map[string]string) map[string]string { return v })
		}
	}

	allSenderName := make([]string, 0)
	for _, v := range cur {
		allSenderName = append(allSenderName, lo.Map(v, func(m map[string]string, _ int) string { return m["name"] })...)
	}
	if dup := lo.FindDuplicates(allSenderName); len(dup) > 0 {
		ctx.AbortWithError(http.StatusBadRequest, fmt.Errorf("duplicate sender name = %v", dup))
		return
	}

	if reflect.DeepEqual(pre, cur) {
		return
	}

	m := lo.MapEntries(cur, func(k string, v []map[string]string) (string, any) { return fmt.Sprintf("senders.%s", k), v })

	if err := k.Load(confmap.Provider(m, "."), nil); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	bs, err := k.Marshal(p)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if err = os.WriteFile("conf/conf.yaml", bs, 0666); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}
}
