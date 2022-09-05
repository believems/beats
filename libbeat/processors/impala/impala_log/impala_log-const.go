package impala_log_parse

import (
	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

const (
	procName   = "impala_log"
	pluginName = "ImpalaLog"
	logName    = "processor." + procName
)

// instanceID is used to assign each instance a unique monitoring namespace.
var instanceID = atomic.MakeUint32(0)

// target_fields:timestamp,domain,host,path,logLevel,eventName,threadName,profile,extend
// config defines the configuration for this processor.
type config struct {
	Field         string `config:"field" validate:"required"`
	Target        string `config:"target_field"`
	OverwriteKeys bool   `config:"overwrite_keys"`
	IgnoreMissing bool   `config:"ignore_missing"`
	IgnoreFailure bool   `config:"ignore_failure"`
	Tag           string `config:"tag"`
}

// timestamp,domain,host,logLevel,eventName,threadName,profile,extend
// defaultConfig will return a config with default values.
func defaultConfig() config {

	return config{
		Field:         "message",
		Target:        procName,
		OverwriteKeys: true,
	}
}

// processor defines a syslog processor.
type processor struct {
	config

	log   *logp.Logger
	stats processorStats
}

// processorStats contains the metrics fields for the syslog processor.
type processorStats struct {
	// Success measures the number of successfully parsed syslog messages.
	Success *monitoring.Int
	// Failure measures the number of occurrences where a message was unable to be parsed.
	Failure *monitoring.Int
	// Missing measures the number of occurrences where an event was missing the required input field.
	Missing *monitoring.Int
}
