package model

import (
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/features"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/labels"
	"github.com/apache/dubbo-kubernetes/pkg/config/visibility"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	networking "istio.io/api/networking/v1alpha3"
	"k8s.io/apimachinery/pkg/types"
)

func (ps *PushContext) mergeSubsetRule(p *consolidatedSubRules, subRuleConfig config.Config, exportToSet sets.Set[visibility.Instance]) {
	rule := subRuleConfig.Spec.(*networking.DestinationRule)
	resolvedHost := host.Name(rule.Host)

	var subRules map[host.Name][]*ConsolidatedSubRule

	if resolvedHost.IsWildCarded() {
		subRules = p.wildcardSubRules
	} else {
		subRules = p.specificSubRules
	}

	if mdrList, exists := subRules[resolvedHost]; exists {
		// `appendSeparately` determines if the incoming destination rule would become a new unique entry in the processedDestRules list.
		appendSeparately := true
		for _, mdr := range mdrList {
			if features.EnableEnhancedSubsetRuleMerge {
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

			existingRule := mdr.rule.Spec.(*networking.DestinationRule)
			bothWithoutSelector := rule.GetWorkloadSelector() == nil && existingRule.GetWorkloadSelector() == nil
			bothWithSelector := existingRule.GetWorkloadSelector() != nil && rule.GetWorkloadSelector() != nil
			selectorsMatch := labels.Instance(existingRule.GetWorkloadSelector().GetMatchLabels()).Equals(rule.GetWorkloadSelector().GetMatchLabels())
			if bothWithSelector && !selectorsMatch {
				// If the new destination rule and the existing one has workload selectors associated with them, skip merging
				// if the selectors do not match
				appendSeparately = true
				continue
			}
			// If both the destination rules are without a workload selector or with matching workload selectors, simply merge them.
			// If the incoming rule has a workload selector, it has to be merged with the existing rules with workload selector, and
			// at the same time added as a unique entry in the processedDestRules.
			if bothWithoutSelector || (bothWithSelector && selectorsMatch) {
				appendSeparately = false
			}

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

			// If there is no top level policy and the incoming rule has top level
			// traffic policy, use the one from the incoming rule.
			if mergedRule.TrafficPolicy == nil && rule.TrafficPolicy != nil {
				mergedRule.TrafficPolicy = rule.TrafficPolicy
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
