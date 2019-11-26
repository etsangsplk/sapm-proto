// Copyright 2019 Splunk, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package translator

import (
	"encoding/json"
	"strconv"
	"time"

	jaegerpb "github.com/jaegertracing/jaeger/model"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/signalfx/golib/v3/trace"

	gen "github.com/signalfx/sapm-proto/gen"
)

const (
	clientKind   = "CLIENT"
	serverKind   = "SERVER"
	producerKind = "PRODUCER"
	consumerKind = "CONSUMER"
)

// SFXToSAPMPostRequest takes a slice spans in the SignalFx format and converts it to a SAPM PostSpansRequest
func SFXToSAPMPostRequest(spans []*trace.Span) *gen.PostSpansRequest {
	sr := &gen.PostSpansRequest{}

	batch := &jaegerpb.Batch{}
	batches := []*jaegerpb.Batch{batch}

	// TODO: create one batch per service/process

	for _, sfxSpan := range spans {
		spanID, err := sfxSpanIDtoJaegerSpanID(sfxSpan.ID)
		if err != nil {
			// drop span with bad span ID
			continue
		}

		traceID, err := sfxTraceIDToJaegerTraceID(sfxSpan.TraceID)
		if err != nil {
			// drop span with bad trace ID
			continue
		}

		span := &jaegerpb.Span{
			SpanID:  spanID,
			TraceID: traceID,
		}

		if sfxSpan.Name != nil {
			span.OperationName = *sfxSpan.Name
		}

		if sfxSpan.Duration != nil {
			span.Duration = durationFromMicroseconds(*sfxSpan.Duration)
		}

		if sfxSpan.Timestamp != nil {
			span.StartTime = timeFromMicrosecondsSinceEpoch(*sfxSpan.Timestamp)
		}

		if sfxSpan.Debug != nil && *sfxSpan.Debug {
			span.Flags.SetDebug()
		}

		batch.Process = &jaegerpb.Process{}
		if sfxSpan.LocalEndpoint != nil {
			if sfxSpan.LocalEndpoint.ServiceName != nil {
				batch.Process.ServiceName = *sfxSpan.LocalEndpoint.ServiceName
			}
			if sfxSpan.LocalEndpoint.Ipv4 != nil {
				batch.Process.Tags = append(batch.Process.Tags, jaegerpb.KeyValue{
					Key:   "ip",
					VType: jaegerpb.ValueType_STRING,
					VStr:  *sfxSpan.LocalEndpoint.Ipv4,
				})
			}
		}

		span.Tags = sfxTagsToJaegerTags(sfxSpan.Tags, sfxSpan.RemoteEndpoint, sfxSpan.Kind)

		if sfxSpan.ParentID != nil {
			parentID, err := sfxSpanIDtoJaegerSpanID(*sfxSpan.ParentID)
			if err == nil {
				span.References = append(span.References, jaegerpb.SpanRef{
					TraceID: traceID,
					SpanID:  parentID,
					RefType: jaegerpb.SpanRefType_CHILD_OF,
				})
			}
		}

		span.Logs = sfxAnnotationsToJaegerLogs(sfxSpan.Annotations)

		batch.Spans = append(batch.Spans, span)
	}

	sr.Batches = batches
	return sr
}

func sfxTagsToJaegerTags(tags map[string]string, remoteEndpoint *trace.Endpoint, kind *string) []jaegerpb.KeyValue {
	kv := make([]jaegerpb.KeyValue, 0, len(tags)+4)

	if remoteEndpoint != nil {
		if remoteEndpoint.Ipv4 != nil {
			kv = append(kv, jaegerpb.KeyValue{
				Key:   string(ext.PeerHostIPv4),
				VType: jaegerpb.ValueType_STRING,
				VStr:  *remoteEndpoint.Ipv4,
			})
		}

		if remoteEndpoint.Ipv6 != nil {
			kv = append(kv, jaegerpb.KeyValue{
				Key:   string(ext.PeerHostIPv6),
				VType: jaegerpb.ValueType_STRING,
				VStr:  *remoteEndpoint.Ipv6,
			})
		}

		if remoteEndpoint.Port != nil {
			kv = append(kv, jaegerpb.KeyValue{
				Key:    string(ext.PeerPort),
				VType:  jaegerpb.ValueType_INT64,
				VInt64: int64(*remoteEndpoint.Port),
			})
		}
	}

	if kind != nil {
		kv = append(kv, sfxKindToJaeger(*kind))
	}

	for k, v := range tags {
		kv = append(kv, jaegerpb.KeyValue{
			Key:   k,
			VType: jaegerpb.ValueType_STRING,
			VStr:  v,
		})
	}

	return kv
}

func sfxSpanIDtoJaegerSpanID(id string) (jaegerpb.SpanID, error) {
	i, err := strconv.ParseUint(id, 16, 64)
	return jaegerpb.SpanID(i), err
}

// ensure this works for both 64bit and 128bit IDs passed as strings
func sfxTraceIDToJaegerTraceID(id string) (jaegerpb.TraceID, error) {
	highStr := id[:16]
	lowStr := id[16:]

	traceID := jaegerpb.TraceID{}

	high, err := strconv.ParseUint(highStr, 16, 64)
	if err != nil {
		return traceID, err
	}

	low, err := strconv.ParseUint(lowStr, 16, 64)
	if err != nil {
		return traceID, err
	}

	traceID.High = high
	traceID.Low = low
	return traceID, nil
}

func sfxAnnotationsToJaegerLogs(annotations []*trace.Annotation) []jaegerpb.Log {
	logs := make([]jaegerpb.Log, 0, len(annotations))
	for _, ann := range annotations {
		if ann.Value == nil {
			continue
		}
		log := jaegerpb.Log{}
		if ann.Timestamp != nil {
			log.Timestamp = timeFromMicrosecondsSinceEpoch(*ann.Timestamp)
		}
		var err error
		log.Fields, err = fieldsFromJSONString(*ann.Value)
		if err != nil {
			continue
		}
		logs = append(logs, log)
	}
	return logs
}

func fieldsFromJSONString(jStr string) ([]jaegerpb.KeyValue, error) {
	fields := make(map[string]string)
	kv := make([]jaegerpb.KeyValue, 0, len(fields))
	err := json.Unmarshal([]byte(jStr), &fields)
	if err != nil {
		kv = append(kv, jaegerpb.KeyValue{
			Key:   "event",
			VType: jaegerpb.ValueType_STRING,
			VStr:  jStr,
		})
		return kv, err
	}

	for k, v := range fields {
		kv = append(kv, jaegerpb.KeyValue{
			Key:   k,
			VType: jaegerpb.ValueType_STRING,
			VStr:  v,
		})
	}
	return kv, nil
}

func sfxKindToJaeger(kind string) jaegerpb.KeyValue {
	kv := jaegerpb.KeyValue{
		Key: string(ext.SpanKind),
	}

	switch kind {
	case clientKind:
		kv.VStr = string(ext.SpanKindRPCClientEnum)
	case serverKind:
		kv.VStr = string(ext.SpanKindRPCServerEnum)
	case producerKind:
		kv.VStr = string(ext.SpanKindProducerEnum)
	case consumerKind:
		kv.VStr = string(ext.SpanKindConsumerEnum)
	}
	return kv
}

func durationFromMicroseconds(micros int64) time.Duration {
	return time.Duration(micros) * time.Microsecond
}

func timeFromMicrosecondsSinceEpoch(micros int64) time.Time {
	nanos := micros * int64(time.Microsecond)
	return time.Unix(0, nanos)
}
