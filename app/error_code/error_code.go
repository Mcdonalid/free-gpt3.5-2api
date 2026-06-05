package error_code

import (
	"fmt"
	"net/http"
	"strings"
)

type Error struct {
	Status  int
	Code    int
	Message string
	Detail  string
}

func (e *Error) Error() string {
	return fmt.Sprintf("[%v] %v %v", e.Code, e.Message, e.Detail)
}

func (e *Error) SetDetail(d string) *Error {
	e.Detail = d
	return e
}

var (
	Successful = &Error{Status: http.StatusOK, Code: 0, Message: "successful"}
	Failure    = &Error{Status: http.StatusBadRequest, Code: -1, Message: "failure"}
	GrpcError  = func(message string, detail ...string) *Error {
		return &Error{Status: http.StatusBadRequest, Code: 10001, Message: message, Detail: strings.Join(detail, ", ")}
	}
	NoPermission     = &Error{Status: http.StatusForbidden, Code: 12001, Message: "no permission"}
	TokenExpiration  = &Error{Status: http.StatusUnauthorized, Code: 12002, Message: "token expiration"}
	InvalidToken     = &Error{Status: http.StatusForbidden, Code: 12003, Message: "invalid token"}
	NotFound         = &Error{Status: http.StatusNotFound, Code: 12004, Message: "not found"}
	MethodNotAllowed = &Error{Status: http.StatusMethodNotAllowed, Code: 12005, Message: "method not allowed"}
	RequestTimeout   = &Error{Status: http.StatusRequestTimeout, Code: 12006, Message: "request timeout"}
)

var (
	SignatureVerificationError = &Error{Status: http.StatusPreconditionFailed, Code: 13001, Message: "signature verification error"}
	ExpiredRequest             = &Error{Status: http.StatusBadRequest, Code: 13002, Message: "expired request"}
)

var (
	ParameterError = &Error{Status: http.StatusBadRequest, Code: 14001, Message: "parameter error"}
)

// 用户相关
var (
	UsernameOrPasswordError     = &Error{Status: http.StatusOK, Code: 15001, Message: "用户名或密码错误"}
	CurrentAccountInDisabled    = &Error{Status: http.StatusOK, Code: 15002, Message: "当前账号被禁用"}
	UsernameInUsed              = &Error{Status: http.StatusOK, Code: 15003, Message: "用户名已被使用"}
	PhoneInUsed                 = &Error{Status: http.StatusOK, Code: 15004, Message: "手机号已被使用"}
	TOTPSecurityCodeError       = &Error{Status: http.StatusOK, Code: 15004, Message: "安全码错误"}
	UserNotExist                = &Error{Status: http.StatusOK, Code: 15005, Message: "用户不存在"}
	NoPermissionSetUser         = &Error{Status: http.StatusOK, Code: 15006, Message: "没有权限修改此用户"}
	UserNoneCurrentOrganization = &Error{Status: http.StatusOK, Code: 15007, Message: "用户没有当前组织架构"}
	UserRegisterFailed          = &Error{Status: http.StatusOK, Code: 15008, Message: "用户注册失败"}
	UserLoginFailed             = &Error{Status: http.StatusOK, Code: 15009, Message: "用户登录失败"}
)

var (
	RoleCodeInUsed      = &Error{Status: http.StatusOK, Code: 16001, Message: "角色编码已被使用"}
	RoleNotExist        = &Error{Status: http.StatusOK, Code: 16002, Message: "角色不存在"}
	NoPermissionSetRole = &Error{Status: http.StatusOK, Code: 16003, Message: "没有权限设置此角色"}
)

var (
	OrganizationNotExist        = &Error{Status: http.StatusOK, Code: 17001, Message: "组织架构不存在"}
	NoPermissionSetOrganization = &Error{Status: http.StatusOK, Code: 17002, Message: "没有权限设置此组织架构"}
)

var (
	RouteNotExist = &Error{Status: http.StatusOK, Code: 18001, Message: "路由不存在"}
)

var (
	PermissionNotExist = &Error{Status: http.StatusOK, Code: 19001, Message: "权限不存在"}
)

var (
	CircularReference = &Error{Status: http.StatusOK, Code: 20001, Message: "循环引用"}
)

var (
	QrCodeExpired = &Error{Status: http.StatusOK, Code: 21001, Message: "二维码已过期"}
)

var (
	CaptchaSendFrequently = &Error{Status: http.StatusOK, Code: 22001, Message: "验证码发送过于频繁"}
	CaptchaError          = &Error{Status: http.StatusOK, Code: 22002, Message: "验证码错误"}
)
