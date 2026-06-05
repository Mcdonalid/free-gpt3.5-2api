package result

import (
	"chat2api/app/error_code"
	verify "chat2api/app/verify"
	"chat2api/pkg/logx"
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type JsonBody struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Detail    string `json:"detail,omitempty"`
	RequestID string `json:"request_id,omitempty"`
	Data      any    `json:"data"`
	Spent     string `json:"spent"`
	ctx       *gin.Context
	begin     time.Time
}

/*
JsonBody 用法示例

 1. 最常见：返回成功
    func handler(ctx *gin.Context) {
    jb := result.New(ctx, "demo_action")
    jb.Data = gin.H{"ok": true}
    jb.Successful()
    }

 2. 一步处理 data + err
    data, err := svc.Do(ctx)
    result.New(ctx, "demo_action").AssertSuccessful(data, err)

 3. 统一错误返回
    if err != nil {
    result.New(ctx, "demo_action").Error(err)
    return
    }

 4. 参数绑定与校验
    req := &Req{}
    jb := result.New(ctx, "demo_action")
    if jb.BindJson(req) { // 或 BindQuery(req)
    return // 绑定/校验失败时已自动返回错误响应
    }

 5. 最后统一收口
    data, err := svc.Do(ctx)
    result.New(ctx, "demo_action").Finish(data, err)
*/

func New(ctx *gin.Context, action ...string) *JsonBody {
	if len(action) == 1 {
		ctx.Request = ctx.Request.WithContext(logx.ActionContext(ctx.Request.Context(), action[0]))
	}
	return &JsonBody{ctx: ctx, begin: time.Now()}
}

func (j *JsonBody) Successful() {
	j.Code = error_code.Successful.Code
	j.Message = error_code.Successful.Message
	j.response(error_code.Successful.Status)
}

func (j *JsonBody) AssertSuccessful(data any, err error) {
	if j.AssertError(err) {
		return
	}
	j.Data = data
	j.Successful()
}

func (j *JsonBody) Failure() {
	j.Code = error_code.Failure.Code
	j.Message = error_code.Failure.Message
	j.response(error_code.Failure.Status)
}

func (j *JsonBody) Error(err error, args ...string) {
	if err == nil {
		return
	}

	statusCode := http.StatusInternalServerError

	var e *error_code.Error
	if ok := errors.As(err, &e); ok {
		j.Code = e.Code
		j.Message = e.Message
		if len(args) > 0 {
			j.Detail = strings.Join(args, ", ")
		} else {
			j.Detail = e.Detail
		}
		statusCode = e.Status
	} else if ge, ok := err.(interface{ Message() string }); ok {
		j.Code = error_code.Failure.Code
		j.Message = ge.Message()
	} else {
		j.Code = error_code.Failure.Code
		j.Message = error_code.Failure.Message
		j.Detail = err.Error()
	}

	//buf := make([]byte, 32<<10)
	//buf = buf[:runtime.Stack(buf, false)]
	logx.WithContext(j.ctx.Request.Context()).Errorf("%s %s: %v", j.ctx.Request.Method, j.ctx.Request.RequestURI, err)
	j.response(statusCode)
	j.ctx.Abort()
}

func (j *JsonBody) AssertError(err error) bool {
	if err == nil {
		return false
	}
	j.Error(err)
	return true
}

func (j *JsonBody) ParamError(err error) {
	j.Code = error_code.ParameterError.Code
	j.Message = err.Error()
	logx.WithContext(j.Context()).Error(err)

	j.response(error_code.ParameterError.Status)
}

func (j *JsonBody) AssertParamError(err error) bool {
	if err == nil {
		return false
	}
	j.ParamError(err)
	return true
}

func (j *JsonBody) Context() context.Context {
	return j.ctx.Request.Context()
}

func (j *JsonBody) response(status int) {

	d := time.Since(j.begin)
	if d > time.Minute {
		d = d - d%time.Second
	}
	j.Spent = d.String()
	j.RequestID = strings.TrimSpace(j.ctx.GetHeader("X-Request-ID"))

	j.ctx.JSON(status, j)
}

type IRequest interface {
	BeforeProcessor()
}

func (j *JsonBody) BindQuery(v any) bool {
	if j.AssertError(j.ctx.ShouldBindQuery(v)) {
		return true
	}
	if request, ok := v.(IRequest); ok {
		request.BeforeProcessor()
		v = request
	}
	return j.AssertParamError(verify.Verify(v))
}

func (j *JsonBody) BindJson(v any) bool {
	if j.AssertError(j.ctx.ShouldBindJSON(v)) {
		return true
	}
	if request, ok := v.(IRequest); ok {
		request.BeforeProcessor()
		v = request
	}
	return j.AssertParamError(verify.Verify(v))
}

func (j *JsonBody) Finish(data any, err error) {
	if !j.AssertError(err) {
		j.Data = data
		j.Successful()
	}
}
