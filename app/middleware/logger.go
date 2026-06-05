package middleware

import (
	"chat2api/pkg/logx"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const requestIDHeader = "X-Request-ID"

func Logger() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		requestID := ensureRequestID(ctx)
		reqCtx := logx.TagContext(ctx.Request.Context(), "request")
		reqCtx = logx.TraceIdContext(reqCtx, requestID)
		ctx.Request = ctx.Request.WithContext(reqCtx)
		start := time.Now()
		ctx.Next()
		spent := time.Since(start)
		logx.WithContext(ctx.Request.Context()).WithFields(logx.Fields{
			"client":  ctx.ClientIP(),
			"method":  ctx.Request.Method,
			"status":  ctx.Writer.Status(),
			"latency": spent.Truncate(time.Microsecond).String(),
			"path":    ctx.Request.RequestURI,
		}).Info("request completed")
	}
}

func ensureRequestID(ctx *gin.Context) string {
	// request_id 是前端、网关、operation_log 和应用日志之间的最小串联键。
	// 输入前置条件：调用方可通过 X-Request-ID 传入已有链路 ID；为空时由后端生成。
	// 失败或异常行为：随机数生成失败时回退到时间戳字符串，保证响应和日志里仍有可检索 ID。
	requestID := strings.TrimSpace(ctx.GetHeader(requestIDHeader))
	if requestID == "" {
		requestID = newRequestID()
		ctx.Request.Header.Set(requestIDHeader, requestID)
	}
	ctx.Header(requestIDHeader, requestID)
	return requestID
}

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
