package utils

import "time"

// TimeFormatNow 当前时间格式化字符串
func TimeFormatNow() string {
	return TimeFormat(time.Now())
}

// TimeFormat 按照2006-01-02T15:04:05.999999999Z07:00格式转换输入时间
func TimeFormat(t time.Time) string {
	return t.Format(time.RFC3339Nano)
}
