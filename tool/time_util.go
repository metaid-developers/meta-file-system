package tool

import (
	"time"
)

const hourTime = 3600 * 1000

var (
	l, _   = time.LoadLocation("UTC")
	asL, _ = time.LoadLocation("Asia/Shanghai")
)

func MakeTimestamp() int64 {
	//time.Now().UnixMilli()
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func MakeDate(timestamp int64) string {
	timeFormat := "2006-01-02 15:04:05(UTC)"
	return time.Unix(timestamp/1000, 0).In(l).Format(timeFormat)
}

func MakeTimestampV2(date string) (int64, error) {
	timeFormat := "2006-01-02-15-04-05"
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return 0, err
	}

	t, err := time.ParseInLocation(timeFormat, date, location)
	if err != nil {
		return 0, err
	}
	return t.UnixNano() / int64(time.Millisecond), nil
}
func MakeDateV2(timestamp int64) string {
	timeFormat := "20060102150405(UTC)"
	return time.Unix(timestamp/1000, 0).In(l).Format(timeFormat)
}

//00:00:00-time
func GetToday0Time() int64 {
	currentTime := time.Now()
	startTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, l)
	return startTime.UnixNano() / 1e6
}

//23:59:59-time
func GetToday24Time() int64 {
	currentTime := time.Now()
	endTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 23, 59, 59, 0, l)
	return endTime.UnixNano() / 1e6
}

func Get0And24TimeByTimestamp(timestamp int64) (int64, int64) {
	currentTime := time.Unix(timestamp/1000, 0)
	startTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, l)
	endTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 23, 59, 59, 0, l)
	return startTime.UnixNano() / 1e6, endTime.UnixNano() / 1e6
}
