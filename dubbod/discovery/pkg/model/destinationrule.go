//
// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package model

import (
	networking "github.com/apache/dubbo-kubernetes/api/networking/v1alpha3"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/features"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/visibility"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"k8s.io/apimachinery/pkg/types"
)

func (ps *PushContext) mergeDestinationRule(p *consolidatedSubRules, subRuleConfig config.Config, exportToSet sets.Set[visibility.Instance]) {
	rule := subRuleConfig.Spec.(*networking.DestinationRule)
	resolvedHost := host.Name(rule.Host)

	var subRules map[host.Name][]*ConsolidatedSubRule

	if resolvedHost.IsWildCarded() {
		subRules = p.wildcardSubRules
	} else {
		subRules = p.specificSubRules
	}

	if mdrList, exists := subRules[resolvedHost]; exists {
		log.Infof("found existing rules for host %s (count: %d)", resolvedHost, len(mdrList))
		// `appendSeparately` determines if the incoming destination rule would become a new unique entry in the processedDestRules list.
		appendSeparately := true
		for _, mdr := range mdrList {
			if features.EnableEnhancedDestinationRuleMerge {
				if exportToSet.Equals(mdr.exportTo) {
					appendSeparately = false
				} else if len(mdr.exportTo) > 0 && exportToSet.SupersetOf(mdr.exportTo) {
					// If the new exportTo is superset of existing, merge and also append as a standalone one
					appendSeparately = true
				} else {
					// can not merge with existing one, append as a standalone one
					appendSeparately = true
					continue
				}
			}

			// Merge destination rules for the same host
			appendSeparately = false
			log.Debugf("will merge rules for host %s", resolvedHost)

			// Deep copy destination rule, to prevent mutate it later when merge with a new one.
			// This can happen when there are more than one destination rule of same host in one namespace.
			copied := mdr.rule.DeepCopy()
			mdr.rule = &copied
			mdr.from = append(mdr.from, subRuleConfig.NamespacedName())
			mergedRule := copied.Spec.(*networking.DestinationRule)

			existingSubset := sets.String{}
			for _, subset := range mergedRule.Subsets {
				existingSubset.Insert(subset.Name)
			}
			// we have another destination rule for same host.
			// concatenate both of them -- essentially add subsets from one to other.
			// Note: we only add the subsets and do not overwrite anything else like exportTo or top level
			// traffic policies if they already exist
			for _, subset := range rule.Subsets {
				if !existingSubset.Contains(subset.Name) {
					// if not duplicated, append
					mergedRule.Subsets = append(mergedRule.Subsets, subset)
				}
			}

			// Merge top-level traffic policy. Historically we only copied the first non-nil policy,
			// which meant a later DestinationRule that supplied TLS settings was ignored once a prior
			// rule (e.g. subsets only) existed. To match Dubbo's behavior and ensure Proxyless gRPC
			// can enable mTLS after subsets are defined, allow the incoming rule to override the TLS
			// portion even when a Common TrafficPolicy already exists.
			if rule.TrafficPolicy != nil {
				if mergedRule.TrafficPolicy == nil {
					// First rule with TrafficPolicy, copy it entirely
					mergedRule.TrafficPolicy = rule.TrafficPolicy
					log.Infof("copied TrafficPolicy from new rule to merged rule for host %s (has TLS: %v)",
						resolvedHost, rule.TrafficPolicy.Tls != nil)
				} else {
					// Merge TrafficPolicy fields, with TLS settings from the latest rule taking precedence
					if rule.TrafficPolicy.Tls != nil {
						// TLS settings from the latest rule always win (DUBBO_MUTUAL)
						mergedRule.TrafficPolicy.Tls = rule.TrafficPolicy.Tls
						log.Infof("updated TLS settings in merged TrafficPolicy for host %s (mode: %v)",
							resolvedHost, rule.TrafficPolicy.Tls.Mode)
					}
					// Merge other TrafficPolicy fields if needed (loadBalancer, connectionPool, etc.)
					// For now, we only merge TLS as it's the critical setting for mTLS
				}
			}
		}
		if appendSeparately {
			subRules[resolvedHost] = append(subRules[resolvedHost], ConvertConsolidatedDestRule(&subRuleConfig, exportToSet))
		}
		return
	}
	// DestinationRule does not exist for the resolved host so add it
	subRules[resolvedHost] = append(subRules[resolvedHost], ConvertConsolidatedDestRule(&subRuleConfig, exportToSet))
}

func ConvertConsolidatedDestRule(cfg *config.Config, exportToSet sets.Set[visibility.Instance]) *ConsolidatedSubRule {
	return &ConsolidatedSubRule{
		exportTo: exportToSet,
		rule:     cfg,
		from:     []types.NamespacedName{cfg.NamespacedName()},
	}
}
