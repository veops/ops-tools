package middleware

import (
	"bytes"
	"encoding/json"

	"github.com/gin-gonic/gin"
)

type bodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyWriter) Write(b []byte) (int, error) {
	return w.body.Write(b)
}

func Error2Resp() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		wb := &bodyWriter{
			body:           &bytes.Buffer{},
			ResponseWriter: ctx.Writer,
		}
		ctx.Writer = wb

		ctx.Next()

		obj := make(map[string]any)
		if len(ctx.Errors) <= 0 {
			json.Unmarshal(wb.body.Bytes(), &obj)
			obj["msg"] = "ok"
		} else {
			obj["msg"] = ctx.Errors.String()
		}
		bs, _ := json.Marshal(obj)
		wb.ResponseWriter.Write(bs)
	}
}
