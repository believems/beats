package impala_log_parse

import (
	"fmt"
	"github.com/believems/e4-log"
	"github.com/elastic/beats/v7/libbeat/processors/impala"
	"regexp"
	"strconv"
	"time"
)

//[IWEF]mmdd hh:mm:ss.uuuuuu threadid file:line] msg
//"I0828 10:36:22.172153 25339 statestore.cc:282] Subscriber 'impalad@e2de5bdh002:6001' registered (registration id: c24233er53agg:c89gg6fds880)"

const TimePattern = "2006-01-02 15:04:05"
const LogPatternSize = 10

var LogPattern = regexp.MustCompile(`(?P<log_level>[IWEF])(?P<month>0?[1-9]|1[0-2])(?P<day>(0[1-9])|(([12][0-9])|(3[01])|[1-9]))\s+(?P<time>(2[0-3]|[01][0-9]):([0-5][0-9]):([0-5][0-9]))\.(?P<micro_second>\d{6})\s+(?P<thread_name>\d+)\s+(?P<component>\b\w+\b)\.(?P<line_ext>\b\w+\b):(?P<code_line>\d+)]\s+(?P<msg>.*)`)

func Parse(line string) (*e4_log.E4Log, error) {
	data, err := parseMap(line)
	if err != nil {
		return nil, err
	}
	e4log, err := buildFromMap(data)
	return e4log, err
}

func parseMap(line string) (map[string]string, error) {
	match := LogPattern.FindStringSubmatch(line)
	groupNames := LogPattern.SubexpNames()
	values := make(map[string]string)
	// 转换为map
	for i, name := range groupNames {
		if i != 0 && name != "" { // 第一个分组为空（也就是整个匹配）
			values[name] = match[i]
		}
	}
	return values, nil
}

func buildFromMap(values map[string]string) (*e4_log.E4Log, error) {
	mapSize := len(values)
	if mapSize != LogPatternSize {
		return nil, fmt.Errorf("Log Pattern Size should be %d but %d", LogPatternSize, mapSize)
	}
	logLevel, err := parseLogLevel(values["log_level"])
	if err != nil {
		return nil, err
	}
	timeStr := strconv.Itoa(time.Now().Year()) + "-" + values["month"] + "-" + values["day"] + " " + values["time"]
	tt, err := time.ParseInLocation(TimePattern, timeStr, time.Local)
	if err != nil {
		return nil, err
	}
	msg := values["msg"]
	component := values["component"]
	threadName := values["thread_name"]
	location := component + "." + values["line_ext"] + ":" + values["code_line"]
	host := impala.GetLocalIP()
	return &e4_log.E4Log{
		Timestamp:   tt,
		Host:        host,
		Application: impala.Application,
		Component:   component,
		LogLevel:    logLevel,
		ThreadName:  threadName,
		Extend:      e4_log.E4Extend{"location": location},
		Msg:         msg,
	}, nil
}

func parseLogLevel(str string) (string, error) {
	switch {
	case str == "I":
		return "INFO", nil
	case str == "W":
		return "WARN", nil
	case str == "E":
		return "ERROR", nil
	case str == "F":
		return "FATAL", nil
	default:
		return "", fmt.Errorf("Unknown Log Level: %s", str)
	}
}
