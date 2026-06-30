// Licensed to LinDB under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. LinDB licenses this file to you under
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

package util

import (
	"errors"
	"fmt"

	agentapi "github.com/lindb/linsight/ai/api"
	modelcomp "github.com/lindb/linsight/ai/components/model"
)

// ClassifyError translates a Chat error into a machine-readable SSEErrorCode and a
// user-facing message string. Both the handler layer (SSE delivery) and the service
// layer (DB persistence) use this function so that the stored error message always
// matches what was sent to the client.
func ClassifyError(err error) (agentapi.SSEErrorCode, string) {
	switch {
	case errors.Is(err, agentapi.ErrTooManyRequests):
		return agentapi.SSEErrorRateLimit, "too many concurrent requests, please try again later"
	case errors.Is(err, modelcomp.ErrNoDefaultModel), errors.Is(err, modelcomp.ErrNoEnabledModels):
		return agentapi.SSEErrorModelNotConfigured, "no AI model configured for this org — please add and enable a model in AI Settings"
	default:
		return agentapi.SSEErrorInternal, fmt.Sprintf("internal error: %s", err)
	}
}
