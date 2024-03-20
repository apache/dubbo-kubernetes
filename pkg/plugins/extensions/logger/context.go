/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package logger

import (
	"context"
)

import (
	"go.opentelemetry.io/otel/trace"
)

type spanLogValuesProcessorKey struct{}

// SpanLogValuesProcessor should be a function which process received
// trace.Span. Returned []]interface{} will be later added as logger values.
type SpanLogValuesProcessor func(trace.Span) []interface{}

// NewSpanLogValuesProcessorContext will enrich the provided context with
// the provided spanLogValuesProcessor. It may be useful for any application
// which depends on dubbo, but wants to for example transform trace/span ids
// from otel to datadog format.
func NewSpanLogValuesProcessorContext(
	ctx context.Context,
	fn SpanLogValuesProcessor,
) context.Context {
	return context.WithValue(ctx, spanLogValuesProcessorKey{}, fn)
}

func FromSpanLogValuesProcessorContext(ctx context.Context) (SpanLogValuesProcessor, bool) {
	fn, ok := ctx.Value(spanLogValuesProcessorKey{}).(SpanLogValuesProcessor)
	return fn, ok
}
