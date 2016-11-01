package conslack

import (
	"strconv"
	"time"
)

// formatTimestamp formats a timestamp to string
func formatTimestamp(str string) string {
	ts, _ := strconv.ParseFloat(str, 64)
	t := time.Unix(int64(ts), 0)
	return t.In(time.Local).Format("02/01/2006 15:04")
}
