package log

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func tagToFormat(tag string) string {
	return fmt.Sprintf("${%s}", tag)
}

func buildLog(result io.Writer, format string, lrw *logResponseWriter, r *http.Request) {
	formatLog := strings.ReplaceAll(format, tagToFormat(TagMethod), r.Method)
	formatLog = strings.ReplaceAll(formatLog, tagToFormat(TagPath), r.URL.Path)
	formatLog = strings.ReplaceAll(formatLog, tagToFormat(TagStatus), strconv.Itoa(lrw.statusCode))
	result.Write([]byte(formatLog))
}
