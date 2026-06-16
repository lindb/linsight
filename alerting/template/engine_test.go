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

package template

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestEngine builds a RenderingEngine backed by a static org template string.
// The returned engine always returns orgContent for any orgID (single-org test helper).
func newTestEngine(orgContent string) *RenderingEngine {
	return NewRenderingEngine(func(_ int64) (string, error) {
		return orgContent, nil
	}, time.Minute)
}

// ─── Render / org-block merging ──────────────────────────────────────────────

func TestRenderingEngine_NoDuplicateDefineError(t *testing.T) {
	// Core regression test: org templates and user template both define the same block.
	// Before the fix this caused:
	//   parse template: template: notification:21: multiple definition of template "custom.alerts"
	orgTemplates := `{{define "custom.alerts"}}ORG: {{ .ReceiverName }}{{end}}`
	userTemplate := `{{define "custom.alerts"}}USER: {{ .ReceiverName }}{{end}}{{ template "custom.alerts" . }}`

	engine := newTestEngine(orgTemplates)
	data := &NotificationData{ReceiverName: "slack-ops"}

	result, err := engine.Render(1, userTemplate, data)
	require.NoError(t, err, "duplicate {{define}} must not cause an error")
	assert.Equal(t, "USER: slack-ops", result)
}

func TestRenderingEngine_UserDefineWins(t *testing.T) {
	// When both org and user define the same block, the user's version must win.
	orgTemplates := `{{define "alert.title"}}ORG title{{end}}`
	userTemplate := `{{define "alert.title"}}USER title{{end}}{{ template "alert.title" . }}`

	engine := newTestEngine(orgTemplates)
	result, err := engine.Render(1, userTemplate, &NotificationData{})
	require.NoError(t, err)
	assert.Equal(t, "USER title", result)
}

func TestRenderingEngine_OrgBlocksAvailableAsRef(t *testing.T) {
	// When user template references a block only defined in org templates, it must resolve.
	orgTemplates := `{{define "shared.footer"}}footer content{{end}}`
	userTemplate := `header - {{ template "shared.footer" . }}`

	engine := newTestEngine(orgTemplates)
	result, err := engine.Render(1, userTemplate, &NotificationData{})
	require.NoError(t, err)
	assert.Equal(t, "header - footer content", result)
}

func TestRenderingEngine_NonConflictingOrgBlocks(t *testing.T) {
	// When org and user define different blocks, both must be accessible in the template.
	orgTemplates := `{{define "org.block"}}ORG-BLOCK{{end}}`
	userTemplate := `{{define "user.block"}}USER-BLOCK{{end}}{{ template "user.block" . }}-{{ template "org.block" . }}`

	engine := newTestEngine(orgTemplates)
	result, err := engine.Render(1, userTemplate, &NotificationData{})
	require.NoError(t, err)
	assert.Equal(t, "USER-BLOCK-ORG-BLOCK", result)
}

func TestRenderingEngine_AllDefinesAutoExecuteFirst(t *testing.T) {
	// Verifies that when the template is composed entirely of {{define}} blocks with
	// no root execution content, the first user-defined block is auto-executed so the
	// preview produces visible output (the common org-template authoring pattern).
	userTemplate := `{{- /* comment */ -}}
{{ define "custom.alerts" -}}
{{ len .Alerts }} alert(s)
{{ range .Alerts -}}
  Status: {{ .Status }}
{{ end -}}
{{ end -}}
{{ define "alert.helper" }}helper{{end -}}`

	engine := newTestEngine("")
	data := &NotificationData{
		Status: "firing",
		Alerts: []TemplateAlert{
			{Status: "firing"},
			{Status: "resolved"},
		},
	}

	result, err := engine.Render(1, userTemplate, data)
	require.NoError(t, err)
	// "custom.alerts" is the first defined block and must be auto-executed.
	assert.Contains(t, result, "2 alert(s)")
	assert.Contains(t, result, "Status: firing")
	assert.Contains(t, result, "Status: resolved")
}

// ─── Cache behaviour ─────────────────────────────────────────────────────────

func TestRenderingEngine_Invalidate(t *testing.T) {
	// Invalidate must force a reload on the next Render call.
	var callCount atomic.Int32
	engine := NewRenderingEngine(func(_ int64) (string, error) {
		callCount.Add(1)
		return `{{define "block"}}v` + func() string {
			if callCount.Load() == 1 {
				return "1"
			}
			return "2"
		}() + `{{end}}`, nil
	}, time.Minute)

	// First render — loads and caches.
	_, err := engine.Render(1, `{{ template "block" . }}`, &NotificationData{})
	require.NoError(t, err)
	assert.Equal(t, int32(1), callCount.Load())

	// Second render — should hit cache, no extra load.
	_, err = engine.Render(1, `{{ template "block" . }}`, &NotificationData{})
	require.NoError(t, err)
	assert.Equal(t, int32(1), callCount.Load(), "second render must use cache")

	// After invalidation — must reload.
	engine.Invalidate(1)
	_, err = engine.Render(1, `{{ template "block" . }}`, &NotificationData{})
	require.NoError(t, err)
	assert.Equal(t, int32(2), callCount.Load(), "post-invalidate render must reload")
}

func TestRenderingEngine_LoadFnError(t *testing.T) {
	// When the loader returns an error, Render must propagate it.
	loadErr := errors.New("db unavailable")
	engine := NewRenderingEngine(func(_ int64) (string, error) {
		return "", loadErr
	}, time.Minute)

	_, err := engine.Render(1, "hello", &NotificationData{})
	require.Error(t, err)
	assert.ErrorIs(t, err, loadErr)
}

func TestRenderingEngine_TTLExpiry(t *testing.T) {
	// After TTL expires, the next call must reload from the loader.
	var callCount atomic.Int32
	engine := NewRenderingEngine(func(_ int64) (string, error) {
		callCount.Add(1)
		return "", nil
	}, 50*time.Millisecond) // very short TTL

	_, err := engine.Render(1, "hello", &NotificationData{})
	require.NoError(t, err)
	assert.Equal(t, int32(1), callCount.Load())

	// Wait for cache to expire.
	time.Sleep(100 * time.Millisecond)

	_, err = engine.Render(1, "hello", &NotificationData{})
	require.NoError(t, err)
	assert.Equal(t, int32(2), callCount.Load(), "expired cache must trigger reload")
}

func TestRenderingEngine_ConcurrentAccess(t *testing.T) {
	// Concurrent Render calls on a cold cache must not race (run with -race).
	// The loader is called at most a small number of times — not once per goroutine.
	var callCount atomic.Int32
	engine := NewRenderingEngine(func(_ int64) (string, error) {
		callCount.Add(1)
		return "", nil
	}, time.Minute)

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_, _ = engine.Render(1, "hello", &NotificationData{})
		}()
	}
	wg.Wait()

	// With double-checked locking, the loader fires once (or a very small number of
	// times if goroutines race through the read-lock check simultaneously).
	assert.LessOrEqual(t, callCount.Load(), int32(goroutines),
		"loader must not be called more times than the number of concurrent callers")
}

// ─── RenderNamed ─────────────────────────────────────────────────────────────

func TestRenderingEngine_RenderNamed(t *testing.T) {
	// RenderNamed must execute a named block defined in org templates.
	orgTemplates := `{{define "notify.title"}}[{{ .Status | upper }}] alert{{end}}`
	engine := newTestEngine(orgTemplates)

	result, err := engine.RenderNamed(1, "notify.title", &NotificationData{Status: "firing"})
	require.NoError(t, err)
	assert.Equal(t, "[FIRING] alert", result)
}

func TestRenderingEngine_RenderNamed_UnknownBlock(t *testing.T) {
	// RenderNamed must return an error when the named block does not exist.
	engine := newTestEngine("")
	_, err := engine.RenderNamed(1, "no.such.block", &NotificationData{})
	require.Error(t, err)
}

// ─── NewRenderingEngine guard ─────────────────────────────────────────────────

func TestNewRenderingEngine_NilLoadFnPanics(t *testing.T) {
	// Passing nil loadFn is a programming error and must panic immediately rather
	// than producing a confusing nil-pointer dereference later at render time.
	require.Panics(t, func() {
		NewRenderingEngine(nil, time.Minute)
	})
}
