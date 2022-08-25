// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package impala_profile

import (
	"encoding/json"
	"errors"
	"fmt"
	decoder "github.com/believems/impala-profile-decode"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/checks"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"strconv"
)

// init will register various aspects of this processor.
func init() {
	processors.RegisterPlugin(procName,
		checks.ConfigChecked(New,
			checks.RequireFields(
				"field",
			),
			checks.AllowedFields(
				"field",
				"target_field",
				"overwrite_keys",
				"ignore_missing",
				"ignore_failure",
				"tag",
			),
		),
	)
	jsprocessor.RegisterPlugin(pluginName, New)
}

// timestamp,domain,host,path,logLevel,eventName,threadName,profile,extend
// defaultConfig will return a config with default values.
func defaultConfig() config {

	return config{
		Field:  "message",
		Target: procName,
		Const: mapstr.M{
			"domain":     "Impala",
			"logLevel":   "INFO",
			"eventName":  "Profile",
			"threadName": "MAIN",
		},
		OverwriteKeys: true,
	}
}

// New creates a new processor from the provided configuration, or an error if the configuration is invalid.
func New(c *conf.C) (processors.Processor, error) {
	cfg := defaultConfig()

	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("fail to unpack the "+procName+" processor configuration: %w", err)
	}

	id := int(instanceID.Inc())
	log := logp.NewLogger(logName).With("instance_id", id)
	registryName := logName + "." + strconv.Itoa(id)

	if cfg.Tag != "" {
		log = log.With("tag", cfg.Tag)
		registryName = logName + "." + cfg.Tag + "-" + strconv.Itoa(id)
	}
	registry := monitoring.Default.NewRegistry(registryName, monitoring.DoNotReport)
	return &processor{
		config: cfg,
		log:    log,
		stats: processorStats{
			Success: monitoring.NewInt(registry, "success"),
			Failure: monitoring.NewInt(registry, "failure"),
			Missing: monitoring.NewInt(registry, "missing"),
		},
	}, nil
}

// Run will process an event and will update the fields based on the parsed message, or an error if the
// message could not be parsed. If an error occurs and the configuration is set to not ignore errors,
// the 'error.message' field will be set with error that was encountered.
func (p *processor) Run(event *beat.Event) (*beat.Event, error) {
	if err := p.run(event); err != nil && !p.IgnoreFailure {
		err = fmt.Errorf(procName+" failed to process field %q: %w", p.Field, err)
		appendStringField(event.Fields, "error.message", err.Error())
		return event, err
	}

	return event, nil
}

// run will parse the event and populate fields on the event.
func (p *processor) run(event *beat.Event) error {
	value, err := event.GetValue(p.Field)
	if err != nil {
		if errors.Is(err, mapstr.ErrKeyNotFound) {
			if p.IgnoreMissing {
				return nil
			}
			p.stats.Missing.Inc()
		}
		if !p.IgnoreFailure {
			p.stats.Failure.Inc()
		}
		return err
	}

	data, ok := value.(string)
	if !ok {
		p.stats.Failure.Inc()
		return fmt.Errorf("type of field %q is not a string", p.Field)
	}
	impalaProfile, err := decoder.DecodeImpalaProfileLine(data)
	if err != nil {
		p.stats.Failure.Inc()
	} else {
		p.stats.Success.Inc()
	}
	valueMap, err := impalaProfile.StringMap()
	if err != nil {
		p.stats.Failure.Inc()
	} else {
		p.stats.Success.Inc()
	}
	for k, v := range p.Const {
		valueMap[k] = v
	}
	event.PutValue(p.Target, mapstr.M(valueMap))
	return err
}

// String will return a string representation of this processor (the configuration).
func (p *processor) String() string {
	data, _ := json.Marshal(p.config)

	return procName + "=" + string(data)
}

// appendStringField appends value to field. If field is nil (not present in the map), then
// the resulting field value will be a string. If the existing field is a string, then field
// value will be converted to a string slice. If the existing field is a string slice or
// interface slice, then the new value will be appended. If the existing value is some
// other type, then this function does nothing.
func appendStringField(m mapstr.M, field, value string) {
	v, _ := m.GetValue(field)
	switch t := v.(type) {
	case nil:
		_, _ = m.Put(field, value)
	case string:
		_, _ = m.Put(field, []string{t, value})
	case []string:
		_, _ = m.Put(field, append(t, value))
	case []interface{}:
		_, _ = m.Put(field, append(t, value))
	}
}
