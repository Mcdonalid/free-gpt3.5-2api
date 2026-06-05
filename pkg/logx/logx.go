package logx

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
)

type (
	Logger = logrus.Logger
	Entry  = logrus.Entry
	Hook   = logrus.Hook
	Level  = logrus.Level
	Fields = logrus.Fields
)

func SetLevel(level Level) {
	logrus.SetLevel(level)
}

func SetFormatter(format logrus.Formatter) {
	logrus.SetFormatter(format)
}

func AddHook(hook Hook) {
	logrus.AddHook(hook)
}

const (
	TraceIdKey  = "trace"
	UserIDKey   = "uid"
	GuuIDKey    = "guid"
	SchoolIDKey = "school"
	UsernameKey = "username"
	TagKey      = "tag"
	StackKey    = "stack"
)

type (
	traceIdKey  struct{}
	userIDKey   struct{}
	guuIDKey    struct{}
	usernameKey struct{}
	schoolIdKey struct{}
	tagKey      struct{}
	stackKey    struct{}
	requestKey  struct{}
	responseKey struct{}
	diff1Key    struct{}
	diff2Key    struct{}
	actionKey   struct{}
)

func TraceIdContext(ctx context.Context, traceId string) context.Context {
	return context.WithValue(ctx, traceIdKey{}, traceId)
}

func FromTraceIdContext(ctx context.Context) string {
	if v := ctx.Value(traceIdKey{}); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func UserIDContext(ctx context.Context, userId int) context.Context {
	return context.WithValue(ctx, userIDKey{}, userId)
}

func FromUserIDContext(ctx context.Context) int {
	if v := ctx.Value(userIDKey{}); v != nil {
		if s, ok := v.(int); ok {
			return s
		}
	}
	return 0
}

func GuuIDContext(ctx context.Context, userId string) context.Context {
	return context.WithValue(ctx, guuIDKey{}, userId)
}

func FromGuuIDContext(ctx context.Context) string {
	if v := ctx.Value(guuIDKey{}); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func SchoolIDContext(ctx context.Context, userId int) context.Context {
	return context.WithValue(ctx, schoolIdKey{}, userId)
}

func FromSchoolIDContext(ctx context.Context) int {
	if v := ctx.Value(schoolIdKey{}); v != nil {
		if s, ok := v.(int); ok {
			return s
		}
	}
	return 0
}

func UsernameContext(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, usernameKey{}, username)
}

func FromUsernameContext(ctx context.Context) string {
	if v := ctx.Value(usernameKey{}); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func TagContext(ctx context.Context, tag string) context.Context {
	return context.WithValue(ctx, tagKey{}, tag)
}

func FromTagContext(ctx context.Context) string {
	if v := ctx.Value(tagKey{}); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func ActionContext(ctx context.Context, action string) context.Context {
	return context.WithValue(ctx, actionKey{}, action)
}

func FromActionContext(ctx context.Context) string {
	if v := ctx.Value(actionKey{}); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func StackContext(ctx context.Context, stack error) context.Context {
	return context.WithValue(ctx, stackKey{}, stack)
}

func FromStackContext(ctx context.Context) error {
	if v := ctx.Value(tagKey{}); v != nil {
		if e, ok := v.(error); ok {
			return e
		}
	}
	return nil
}

func RequestContext(ctx context.Context, data []byte) context.Context {
	return context.WithValue(ctx, requestKey{}, string(data))
}

func FromRequestContext(ctx context.Context) string {
	if v := ctx.Value(requestKey{}); v != "" {
		if e, ok := v.(string); ok {
			return e
		}
	}
	return ""
}

func ResponseContext(ctx context.Context, data any) context.Context {
	bts, _ := json.Marshal(&data)
	return context.WithValue(ctx, responseKey{}, string(bts))
}

func FromResponseContext(ctx context.Context) string {
	if v := ctx.Value(responseKey{}); v != "" {
		if e, ok := v.(string); ok {
			return e
		}
	}
	return ""
}

func DiffContext(ctx context.Context, data1, data2 any) context.Context {
	ctx = context.WithValue(ctx, diff1Key{}, data1)
	ctx = context.WithValue(ctx, diff2Key{}, data2)
	return ctx
}

func FromDiffContext(ctx context.Context) (any, any) {
	return ctx.Value(diff1Key{}), ctx.Value(diff2Key{})
}

func WithContext(ctx context.Context) *Entry {
	fields := logrus.Fields{}
	if v := FromTraceIdContext(ctx); v != "" {
		fields[TraceIdKey] = v
	}

	if v := FromUserIDContext(ctx); v != 0 {
		fields[UserIDKey] = v
	}

	if v := FromGuuIDContext(ctx); v != "" {
		fields[GuuIDKey] = v
	}

	if v := FromSchoolIDContext(ctx); v != 0 {
		fields[SchoolIDKey] = v
	}

	if v := FromUsernameContext(ctx); v != "" {
		fields[UsernameKey] = v
	}

	if v := FromTagContext(ctx); v != "" {
		fields[TagKey] = v
	}

	if v := FromStackContext(ctx); v != nil {
		fields[StackKey] = fmt.Sprintf("%+v", v)
	}

	return logrus.WithContext(ctx).WithFields(fields)
}

var (
	Tracef          = logrus.Tracef
	Debugf          = logrus.Debugf
	Infof           = logrus.Infof
	Warnf           = logrus.Warnf
	Errorf          = logrus.Errorf
	Fatalf          = logrus.Fatalf
	Panicf          = logrus.Panicf
	Printf          = logrus.Printf
	SetOutput       = logrus.SetOutput
	SetReportCaller = logrus.SetReportCaller
	StandardLogger  = logrus.StandardLogger
	ParseLevel      = logrus.ParseLevel
)
