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

package impala_log_parse

import (
	"fmt"
	"github.com/elastic/beats/v7/libbeat/processors/impala"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	rawLine = "I0828 10:36:22.172153 25339 statestore.cc:282] Subscriber 'impalad@e2de5bdh002:6001' registered (registration id: c24233er53agg:c89gg6fds880)"
)

var timestamp = strconv.Itoa(time.Now().Year()) + "-08-28T10:36:22+08:00"
var impalaLogCases = map[string]struct {
	cfg      *conf.C
	in       mapstr.M
	want     mapstr.M
	wantTime time.Time
	wantErr  bool
}{
	"impala_log_default": {
		cfg: conf.MustNewConfigFrom(mapstr.M{}),
		in: mapstr.M{
			"message": rawLine,
		},
		want: mapstr.M{
			"impala_log": map[string]interface{}{
				"timestamp":   timestamp,
				"application": "Impala",
				"component":   "statestore",
				"host":        impala.GetLocalIP(),
				"msg":         "Subscriber 'impalad@e2de5bdh002:6001' registered (registration id: c24233er53agg:c89gg6fds880)",
				"log_level":   "INFO",
				"thread_name": "25339",
				"extend": map[string]interface{}{
					"location": "statestore.cc:282",
				},
			},
			"message": rawLine,
		},
	},
}

func TestParse(t *testing.T) {
	values, _ := parseMap(rawLine)
	for k, v := range values {
		fmt.Printf("%+15s: %s\n", k, v)
	}
}

func TestTime(t *testing.T) {
	layout := "2006-01-02 15:04:05"
	str := "2022-08-28 10:36:22"
	tt, err := time.ParseInLocation(layout, str, time.Local)

	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(tt)
	fmt.Println(tt.Format(layout))
}

func TestImpalaLog(t *testing.T) {
	for name, tc := range impalaLogCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			p, err := New(tc.cfg)
			if err != nil {
				panic(err)
			}
			event := &beat.Event{
				Fields: tc.in,
			}

			got, gotErr := p.Run(event)
			if tc.wantErr {
				assert.Error(t, gotErr)
			} else {
				assert.NoError(t, gotErr)
			}

			assert.Equal(t, tc.want, got.Fields)
		})
	}
}

func BenchmarkImpalaLog(b *testing.B) {
	for name, bc := range impalaLogCases {
		bc := bc
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {

				p, _ := New(bc.cfg)
				event := &beat.Event{
					Fields: bc.in,
				}

				_, _ = p.Run(event)
			}
		})
	}
}

func TestAppendStringField(t *testing.T) {
	tests := map[string]struct {
		inMap   mapstr.M
		inField string
		inValue string
		want    mapstr.M
	}{
		"nil": {
			inMap:   mapstr.M{},
			inField: "error",
			inValue: "foo",
			want: mapstr.M{
				"error": "foo",
			},
		},
		"string": {
			inMap: mapstr.M{
				"error": "foo",
			},
			inField: "error",
			inValue: "bar",
			want: mapstr.M{
				"error": []string{"foo", "bar"},
			},
		},
		"string-slice": {
			inMap: mapstr.M{
				"error": []string{"foo", "bar"},
			},
			inField: "error",
			inValue: "some value",
			want: mapstr.M{
				"error": []string{"foo", "bar", "some value"},
			},
		},
		"interface-slice": {
			inMap: mapstr.M{
				"error": []interface{}{"foo", "bar"},
			},
			inField: "error",
			inValue: "some value",
			want: mapstr.M{
				"error": []interface{}{"foo", "bar", "some value"},
			},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			appendStringField(tc.inMap, tc.inField, tc.inValue)

			assert.Equal(t, tc.want, tc.inMap)
		})
	}
}
