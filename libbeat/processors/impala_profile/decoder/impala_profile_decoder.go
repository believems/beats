package decoder

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"github.com/apache/thrift/lib/go/thrift"
	"github.com/believems/impala-thrift/runtimeprofile"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DecodeImpalaProfileLine DecodeLine 解析Impala日志行/*
func DecodeImpalaProfileLine(line string) (*ImpalaProfile, error) {
	fields := strings.Fields(line)
	ts := time.Now().Unix()
	queryId := ""
	dataCompressed := ""
	impalaProfile := ""
	var err error
	if len(fields) == 3 {
		queryId = fields[1]
		ts, err = strconv.ParseInt(fields[0], 10, 64)
		dataCompressed = fields[2]
		if err != nil {
			return nil, fmt.Errorf("Wrong Long format[%s] with error: %e", fields[0], err)
		}
	} else if len(fields) == 1 {
		dataCompressed = fields[0]
	} else {
		return nil, fmt.Errorf("Wrong format line: %s with error: %e", line, err)
	}
	queryId, impalaProfile, err = decodeImpalaProfile(dataCompressed)
	if err != nil {
		return nil, fmt.Errorf("decode error:%e", err)
	}

	profile := &ImpalaProfile{}
	profile.QueryId = queryId
	profile.Profile = impalaProfile
	profile.Timestamp = time.Unix(0, ts*int64(time.Millisecond))
	return profile, nil
}

// 进行zlib解压缩
func doZlibUnCompress(compressSrc []byte) ([]byte, error) {
	b := bytes.NewReader(compressSrc)
	var out bytes.Buffer
	r, _ := zlib.NewReader(b)
	_, err := io.Copy(&out, r)
	return out.Bytes(), err
}

func decodeImpalaProfile(dataCompressed string) (string, string, error) {
	possiblyCompressed, err := base64.StdEncoding.DecodeString(dataCompressed)
	if err != nil {
		return "", "", err
	}
	uncompressedBytes, err := doZlibUnCompress(possiblyCompressed)
	if err != nil {
		return "", "", err
	}
	runtimeProfile := &runtimeprofile.TRuntimeProfileTree{}
	transport := thrift.NewTMemoryBufferLen(1024)
	protocol := thrift.NewTCompactProtocolFactory().GetProtocol(transport)
	if err != nil {
		return "", "", err
	}
	deserializer := &thrift.TDeserializer{
		Transport: transport,
		Protocol:  protocol,
	}
	err = deserializer.Read(runtimeProfile, uncompressedBytes)
	if err != nil {
		return "", "", err
	}
	return getQueryIdFromRuntimeProfile(*runtimeProfile), runtimeProfile.String(), err
}

func getQueryIdFromRuntimeProfile(runtimeProfile runtimeprofile.TRuntimeProfileTree) string {
	if len(runtimeProfile.Nodes) > 0 {
		targetTree := runtimeProfile.Nodes[0]
		compileRegex := regexp.MustCompile("\\(id=(.*?)\\)") // 兼容英文括号并取消括号的转义，例如：华南地区 (广州) -> 广州。
		matchArr := compileRegex.FindStringSubmatch(targetTree.Name)
		if len(matchArr) > 0 {
			return matchArr[len(matchArr)-1]
		}
		return ""
	} else {
		return ""
	}
}
