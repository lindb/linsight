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

package manager

import (
	"encoding/json"
	"regexp"

	"github.com/lindb/linsight/model"
)

// traverseNode is the recursive core of the alert routing algorithm.
// ancestors holds the route chain from root to the current node's parent (used for inheritance).
//
// Traversal rules (mirrors Grafana / Alertmanager behaviour):
//   - The root node (ParentUID="") is entered unconditionally.
//   - Child nodes are checked in order; if continue=false on a match, sibling evaluation stops.
//   - When no child matches, the current node is the fallback receiver.
//   - Zero-value timing fields on a leaf are filled from the nearest ancestor.
func traverseNode(node *model.AlertRouteNode, labels map[string]string, ancestors []*model.AlertRoute) []*model.AlertRoute {
	if node == nil {
		return nil
	}
	// Non-root nodes must match the event labels; root (ParentUID="") always passes.
	if node.ParentUID != "" {
		matchers, err := node.ToMatchers()
		if err != nil || !matchAll(labels, matchers) {
			return nil
		}
	}

	newAncestors := append(ancestors, &node.AlertRoute) //nolint:gocritic // intentional accumulation per call stack frame

	// Evaluate children in sort order
	var result []*model.AlertRoute
	childMatched := false
	for _, child := range node.Routes {
		childResults := traverseNode(child, labels, newAncestors)
		if len(childResults) > 0 {
			result = append(result, childResults...)
			childMatched = true
			if !child.Continue {
				// Stop checking siblings once the first non-continue match is found
				break
			}
		}
	}

	// When no child matched, the current node is the fallback/leaf receiver.
	// Inherit zero-value timing fields from ancestors so every dispatched route
	// has meaningful GroupWait / GroupInterval / RepeatInterval / GroupBy values.
	if !childMatched {
		resolved := resolveInheritance(&node.AlertRoute, ancestors)
		result = append(result, resolved)
	}
	return result
}

// resolveInheritance returns a copy of route with zero-value fields filled from the nearest
// ancestor that defines them (walking from closest parent toward root).
func resolveInheritance(route *model.AlertRoute, ancestors []*model.AlertRoute) *model.AlertRoute {
	resolved := *route
	for i := len(ancestors) - 1; i >= 0; i-- {
		p := ancestors[i]
		if resolved.GroupWait == 0 {
			resolved.GroupWait = p.GroupWait
		}
		if resolved.GroupInterval == 0 {
			resolved.GroupInterval = p.GroupInterval
		}
		if resolved.RepeatInterval == 0 {
			resolved.RepeatInterval = p.RepeatInterval
		}
		if len(resolved.GroupBy) == 0 {
			resolved.GroupBy = p.GroupBy
		}
	}
	return &resolved
}

// matchAll returns true when every matcher in the list matches the label set.
func matchAll(labels map[string]string, matchers []model.LabelMatcher) bool {
	for _, m := range matchers {
		val := labels[m.Name]
		switch m.Type {
		case model.MatchEqual:
			if val != m.Value {
				return false
			}
		case model.MatchNotEqual:
			if val == m.Value {
				return false
			}
		case model.MatchRegexp:
			ok, _ := regexp.MatchString("^(?:"+m.Value+")$", val)
			if !ok {
				return false
			}
		case model.MatchNotRegexp:
			ok, _ := regexp.MatchString("^(?:"+m.Value+")$", val)
			if ok {
				return false
			}
		}
	}
	return true
}

// parseEventLabels decodes the JSON labels of an AlertEvent into a map.
func parseEventLabels(ev *model.AlertEvent) map[string]string {
	if len(ev.Labels) == 0 {
		return nil
	}
	result := make(map[string]string)
	_ = json.Unmarshal(ev.Labels, &result)
	return result
}

// extractMatchers decodes a datatypes.JSON field into a []LabelMatcher slice.
func extractMatchers(raw interface{ MarshalJSON() ([]byte, error) }) []model.LabelMatcher {
	data, err := raw.MarshalJSON()
	if err != nil {
		return nil
	}
	var ms []model.LabelMatcher
	_ = json.Unmarshal(data, &ms)
	return ms
}
