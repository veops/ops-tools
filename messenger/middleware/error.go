package middleware

import "github.com/gin-gonic/gin"

func Error2Resp() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Next()

		if len(ctx.Errors) <= 0 {
			return
		}
		ctx.JSON(0, map[string]string{"msg": ctx.Errors.String()})
	}
}
