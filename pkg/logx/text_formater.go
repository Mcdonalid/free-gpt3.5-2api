package logx

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

type TextFormatter struct {
}

func (f *TextFormatter) Format(entry *Entry) ([]byte, error) {
	//前景色   背景色
	// 30  	   40	  黑色
	// 31  	   41	  红色
	// 32  	   42	  绿色
	// 33  	   43     黄色
	// 34  	   44     蓝色
	// 35  	   45 	  紫色
	// 36  	   46 	  青色
	// 37  	   47	  白色
	var color int
	switch entry.Level {
	case logrus.DebugLevel, logrus.TraceLevel:
		color = 37
	case logrus.WarnLevel:
		color = 33
	case logrus.ErrorLevel:
		color = 31
	case logrus.InfoLevel:
		color = 36
	case logrus.FatalLevel:
		color = 41
	case logrus.PanicLevel:
		color = 44
	default:
		color = 36
	}

	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	if entry.HasCaller() {
		_, _ = fmt.Fprintf(b, "\u001B[%dmFunc\u001B[0m %v():%d\n", color, entry.Caller.Function, entry.Caller.Line)
	}
	//entry.Message = strings.TrimSuffix(entry.Message, "\n")
	_, _ = fmt.Fprintf(b, "%s \x1b[%dm%s\x1b[0m", entry.Time.Format("2006-01-02 15:04:05:06"),
		color, func() string {
			level := strings.ToUpper(entry.Level.String())
			if len(level) < 7 {
				for i := len(level); i < 7; i++ {
					level += " "
				}
			}
			return level
		}())

	// 固定输出 tag 区域宽度，避免 REQUEST/GORM 等不同长度 tag 因制表符跳位导致后续消息列不对齐。
	// 为什么这样做：
	// - 终端里 \t 按 tab stop 展开，tag 长度变化会让 message 起始列抖动；
	// - 改为固定宽度后，各类日志消息正文都从同一列开始，便于肉眼快速扫描。
	// 输入与前置条件：
	// - 若上下文未设置 tag，使用 "-" 占位；
	// - tag 统一转大写并限制在固定宽度内显示。
	// 失败或异常行为：
	// - 无外部依赖，不会返回错误；超长 tag 由 fmt 宽度自然扩展，不影响日志输出。
	tagText := "-"
	if tag, ok := entry.Data[TagKey]; ok {
		tagText = strings.ToUpper(fmt.Sprintf("%v", tag))
		delete(entry.Data, TagKey)
	}
	_, _ = fmt.Fprintf(b, "\u001B[%dm%-8s\u001B[0m ", 31, tagText)

	_, _ = fmt.Fprintf(b, " %s", entry.Message)

	if schoolId, ok := entry.Data[SchoolIDKey]; ok {
		_, _ = fmt.Fprintf(b, "[\u001B[%dm%v\u001B[0m]\t", 35, schoolId)
		delete(entry.Data, SchoolIDKey)
	}

	{
		userId, uidOk := entry.Data[UserIDKey]
		username, unameOk := entry.Data[UsernameKey]
		if uidOk && unameOk {
			// USER_ID: USERNAME:
			_, _ = fmt.Fprintf(b, "[\u001B[%dm%v(%v)\u001B[0m]\t", 36, username, userId)
			delete(entry.Data, UserIDKey)
			delete(entry.Data, UsernameKey)
		}
	}

	if gUuid, ok := entry.Data[GuuIDKey]; ok {
		// TAG:
		_, _ = fmt.Fprintf(b, "\u001B[%dm%s\u001B[0m\t", 36, gUuid)
		delete(entry.Data, GuuIDKey)
	}

	if traceId, ok := entry.Data[TraceIdKey]; ok {
		_, _ = fmt.Fprintf(b, " trace=\u001B[%dm%v\u001B[0m", 34, traceId)
		delete(entry.Data, TraceIdKey)
	}

	if stack, ok := entry.Data[StackKey]; ok {
		_, _ = fmt.Fprintf(b, "\u001B[%dm%s\u001B[0m\t", 33, stack)
		delete(entry.Data, StackKey)
	}

	if len(entry.Data) > 0 {
		writeField := func(key string) {
			if value, ok := entry.Data[key]; ok {
				_, _ = fmt.Fprintf(b, " %s=%v", key, value)
				delete(entry.Data, key)
			}
		}
		writeField("client")
		writeField("method")
		writeField("status")
		writeField("latency")
		writeField("path")
		for key, value := range entry.Data {
			_, _ = fmt.Fprintf(b, " %s=%v", key, value)
		}
	}

	b.WriteByte('\n')
	return b.Bytes(), nil
}
