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

package v1alpha1

import (
	"encoding/json"
	"math"
	"sort"
	"strings"
)

import (
	"github.com/pkg/errors"

	"sigs.k8s.io/yaml"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/core/consts"
	"github.com/apache/dubbo-kubernetes/pkg/util/proto"
)

func (x *ConditionRoute) GetVersion() string {
	if x.ToConditionRouteV3() != nil {
		return consts.ConfiguratorVersionV3
	} else {
		return consts.ConfiguratorVersionV3x1
	}
}

func (x *ConditionRoute) ToYAML() ([]byte, error) {
	if msg := x.ToConditionRouteV3x1(); msg != nil {
		return proto.ToYAML(msg)
	} else if msg := x.ToConditionRouteV3(); msg != nil {
		return proto.ToYAML(msg)
	}
	return nil, errors.New(`ConditionRoute validation failed`)
}

func ConditionRouteDecodeFromYAML(content []byte) (*ConditionRoute, error) {
	jsonContent, err := yaml.YAMLToJSON(content)
	if err != nil {
		return nil, err
	}

	_map := map[string]interface{}{}
	err = json.Unmarshal(jsonContent, &_map)
	if err != nil {
		return nil, err
	}

	version := _map[consts.ConfigVersionKey]
	if version == consts.ConfiguratorVersionV3 {
		v3 := new(ConditionRouteV3)
		if err = proto.FromJSON(jsonContent, v3); err != nil {
			return nil, err
		}
		return v3.ToConditionRoute(), nil
	} else if version == consts.ConfiguratorVersionV3x1 {
		v3x1 := new(ConditionRouteV3X1)
		if err = proto.FromJSON(jsonContent, v3x1); err != nil {
			return nil, err
		}
		return v3x1.ToConditionRoute(), nil
	} else {
		return nil, errors.New("invalid condition route format")
	}
}

func (x *ConditionRouteV3) ToConditionRoute() *ConditionRoute {
	return &ConditionRoute{Conditions: &ConditionRoute_ConditionsV3{ConditionsV3: x}}
}

func (x *ConditionRouteV3X1) ToConditionRoute() *ConditionRoute {
	return &ConditionRoute{Conditions: &ConditionRoute_ConditionsV3X1{ConditionsV3X1: x}}
}

func (x *ConditionRoute) ToConditionRouteV3() *ConditionRouteV3 {
	if v, ok := x.Conditions.(*ConditionRoute_ConditionsV3); ok {
		return v.ConditionsV3
	}
	return nil
}

func (x *ConditionRoute) ToConditionRouteV3x1() *ConditionRouteV3X1 {
	if v, ok := x.Conditions.(*ConditionRoute_ConditionsV3X1); ok {
		return v.ConditionsV3X1
	}
	return nil
}

func (x *ConditionRouteV3X1) ReGenerateCondition() {
	if x.XGenerateByCp == nil {
		return
	}
	newCond := x.ListUnGenConditions()

	if x.XGenerateByCp.RegionPrioritize { // add region prioritize logic to userDefinedRule
		regionPrioritizeRules := make([]*ConditionRule, 0, len(newCond))
		for _, rule := range newCond {
			if rule.TrafficDisable {
				continue
			}
			regionPrioritizeRule := &ConditionRule{
				Priority:      rule.Priority + 1,
				From:          rule.From,
				To:            make([]*ConditionRuleTo, 0, len(rule.To)),
				Ratio:         max(rule.Ratio, x.XGenerateByCp.RegionPrioritizeRate),
				Force:         false,
				XGenerateByCp: true,
			}
			for _, to := range rule.To {
				and := ""
				if to.Match != "" {
					and = " & "
				}
				regionPrioritizeRule.To = append(regionPrioritizeRule.To, &ConditionRuleTo{
					Match:  "region=$region" + and + to.Match,
					Weight: to.Weight,
				})
			}
			regionPrioritizeRules = append(regionPrioritizeRules, regionPrioritizeRule)
		}
		newCond = append(newCond, regionPrioritizeRules...)

		if !x.Force { // add failBack logic, upper match all fail, try match this
			failBackCond := &ConditionRule{
				Priority:       0, // Last Match
				From:           &ConditionRuleFrom{Match: "" /* match all */},
				TrafficDisable: false,
				To:             []*ConditionRuleTo{{Match: "region=$region"}},
				Ratio:          x.XGenerateByCp.RegionPrioritizeRate,
				Force:          false,
				XGenerateByCp:  true,
			}
			isExist := false
			for _, rule := range newCond {
				if rule.equal(failBackCond) {
					isExist = true
					break
				}
			}
			if !isExist {
				newCond = append(newCond)
			}
		}
	}

	if x.XGenerateByCp.DisabledIP != nil && len(x.XGenerateByCp.DisabledIP) != 0 { // add traffic disable logic in condition rule
		disableRule := &ConditionRule{
			Priority: math.MaxInt32,
			From: &ConditionRuleFrom{
				Match: "host = " + strings.Join(x.XGenerateByCp.DisabledIP, ","),
			},
			TrafficDisable: true,
			XGenerateByCp:  true,
		}
		newCond = append(newCond, disableRule)
	}
	x.Conditions = newCond
	x.SortConditions()
}

func (x *ConditionRule) equal(cond *ConditionRule) bool {
	if x.Priority != cond.Priority ||
		x.TrafficDisable != cond.TrafficDisable ||
		x.Force != cond.Force ||
		x.Ratio != cond.Ratio {
		return false
	}

	if (x.From != nil && cond.From != nil) && (x.From.Match != cond.From.Match) {
		return false
	}

	if (x.To == nil) != (cond.To == nil) || len(x.To) != len(cond.To) {
		return false
	}

	createKey := func(s string, i int32) interface{} {
		return struct {
			S string
			I int32
		}{s, i}
	}

	set := make(map[interface{}]struct{}, len(cond.To))
	for _, to := range cond.To {
		set[createKey(to.Match, to.Weight)] = struct{}{}
	}

	for _, to := range x.To {
		delete(set, createKey(to.Match, to.Weight))
	}

	return len(set) == 0
}

func (x *ConditionRouteV3X1) SortConditions() {
	sort.Slice(x.Conditions, func(i, j int) bool {
		if x.Conditions[i].TrafficDisable {
			return true
		}
		return x.Conditions[i].Priority > x.Conditions[j].Priority
	})
}

func (x *ConditionRouteV3X1) ListUnGenConditions() []*ConditionRule {
	res := make([]*ConditionRule, 0)
	for _, condition := range x.Conditions {
		if !condition.XGenerateByCp {
			res = append(res, condition)
		}
	}
	return res
}

func (x *ConditionRouteV3X1) ListGenConditions() []*ConditionRule {
	res := make([]*ConditionRule, 0)
	for _, condition := range x.Conditions {
		if condition.XGenerateByCp {
			res = append(res, condition)
		}
	}
	return res
}

func (x *ConditionRouteV3X1) RangeConditions(f func(r *ConditionRule) (isStop bool)) {
	if f == nil {
		return
	}
	for _, condition := range x.Conditions {
		if f(condition) {
			break
		}
	}
}

func (x *ConditionRouteV3X1) RangeConditionsToRemove(f func(r *ConditionRule) (isRemove bool)) {
	if f == nil {
		return
	}
	res := make([]*ConditionRule, len(x.Conditions)/2+1)
	for _, condition := range x.Conditions {
		if !f(condition) {
			res = append(res, condition)
		}
	}
	x.Conditions = res
}

func (x *ConditionRule) IsMatchMethod() (string, bool) {
	conditions := strings.Split(x.From.Match, "&")
	for _, condition := range conditions {
		if idx := strings.Index(condition, "method"); idx != -1 {
			args := strings.Split(condition, consts.Equal)
			if len(args) == 2 {
				return strings.TrimSpace(args[1]), true
			}
		}
	}
	return "", false
}
