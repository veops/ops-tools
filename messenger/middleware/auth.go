package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/samber/lo"
	"github.com/spf13/cast"
)

func Auth(confs []map[string]string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if len(confs) <= 0 {
			ctx.Next()
			return
		}
		for _, conf := range confs {
			t := cast.ToString(conf["type"])
			if type2auth[t] == nil || !type2auth[t](conf, ctx) {
				continue
			}
			ctx.Next()
			return
		}
		ctx.AbortWithStatus(http.StatusUnauthorized)
	}
}

var (
	type2auth = map[string]func(map[string]string, *gin.Context) bool{
		"ip":    authByIP,
		"token": authByToken,
		"sign":  authBySign,
	}
)

func authByIP(conf map[string]string, ctx *gin.Context) bool {
	m, err := filepath.Match(conf["pattern"], ctx.ClientIP())
	return m && err == nil
}

func authByToken(conf map[string]string, ctx *gin.Context) bool {
	return conf["token"] != "" && conf["token"] == ctx.GetHeader("X-Token")
}

func authBySign(conf map[string]string, ctx *gin.Context) bool {
	ts := cast.ToInt64(ctx.GetHeader("X-TS"))
	if ts == 0 || time.Unix(ts, 0).Add(time.Second*60).Before(time.Now()) {
		return false
	}

	body := make(map[string]string)
	if ctx.ShouldBindBodyWith(&body, binding.JSON) != nil {
		return false
	}

	body["nonce"] = ctx.GetHeader("X-Nonce")
	body["ts"] = ctx.GetHeader("X-TS")

	keys := lo.Keys(body)
	sort.Strings(keys)
	kvStr := strings.Join(lo.Map(keys, func(k string, _ int) string { return fmt.Sprintf("%s%s", k, body[k]) }), "")

	mac := hmac.New(sha256.New, []byte(conf["secret"]))
	_, _ = mac.Write([]byte(kvStr))
	return ctx.GetHeader("X-Sign") == base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
