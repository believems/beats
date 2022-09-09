package impala_log_parse

import (
	"fmt"
	"github.com/believems/e4-log"
	"github.com/elastic/beats/v7/libbeat/processors/impala"
	"github.com/vjeantet/grok"
	"strconv"
	"time"
)

//[IWEF]mmdd hh:mm:ss.uuuuuu threadid file:line] msg
//"I0828 10:36:22.172153 25339 statestore.cc:282] Subscriber 'impalad@e2de5bdh002:6001' registered (registration id: c24233er53agg:c89gg6fds880)"

const TimePattern = "2006-01-02 15:04:05"
const LogPatternSize = 10
const LogPattern = "%{IMPALA_LOG_LEVEL:log_level}%{MONTHNUM:month}%{MONTHDAY:day} %{TIME:time}\\.%{MICRO_SECOND:micro_second} %{NUMBER:thread_name} %{WORD:component}\\.%{WORD:line_ext}:%{NUMBER:code_line}] %{GREEDYDATA:msg}"

var grokInstance *grok.Grok = nil

func getInstance() *grok.Grok {
	grokInstance, _ = grok.NewWithConfig(&grok.Config{NamedCapturesOnly: true})
	_ = grokInstance.AddPattern("IMPALA_LOG_LEVEL", "(?:[I|W|E|F]{1})")
	_ = grokInstance.AddPattern("MICRO_SECOND", "(?:[\\d]{6})")
	return grokInstance
}

func Parse(line string) (*e4_log.E4Log, error) {
	data, err := parseMap(line)
	if err != nil {
		return nil, err
	}
	e4log, err := buildFromMap(data)
	return e4log, err
}
func parseMap(line string) (map[string]string, error) {
	values, err := getInstance().Parse(LogPattern, line)
	if err != nil {
		return nil, err
	}
	return values, nil
}

func buildFromMap(values map[string]string) (*e4_log.E4Log, error) {
	mapSize := len(values)
	if mapSize != LogPatternSize {
		return nil, fmt.Errorf("log Pattern Size should be %d but %d", LogPatternSize, mapSize)
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
		return "", fmt.Errorf("unknown Log Level: %s", str)
	}
}
