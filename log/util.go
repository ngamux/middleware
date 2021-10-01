package log

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func tagToFormat(tag string) string {
	return fmt.Sprintf("${%s}", tag)
}

func buildLog(cfg Config, lrw *logResponseWriter, r *http.Request) string {
	formatLog := strings.ReplaceAll(cfg.Format, tagToFormat(TagMethod), r.Method)
	formatLog = strings.ReplaceAll(formatLog, tagToFormat(TagPath), r.URL.Path)
	formatLog = strings.ReplaceAll(formatLog, tagToFormat(TagStatus), strconv.Itoa(lrw.statusCode))
	return formatLog
}
