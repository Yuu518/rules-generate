package output

import (
	"path/filepath"
	"strings"

	"github.com/Yuu518/rules-generate/internal/model"
)

func formatOutputDir(outputDir, format string, splitByFormat bool) string {
	if splitByFormat {
		return filepath.Join(outputDir, format)
	}
	return outputDir
}

func filterLists(ruleMap model.RuleMap, lists []string) model.RuleMap {
	if len(lists) == 0 {
		return ruleMap
	}

	wanted := make(map[string]bool)
	for _, l := range lists {
		wanted[strings.ToLower(strings.TrimSpace(l))] = true
	}

	result := make(model.RuleMap)
	for code, rules := range ruleMap {
		baseName := code
		if idx := strings.Index(code, "@"); idx >= 0 {
			baseName = code[:idx]
		}
		if wanted[code] || wanted[baseName] {
			result[code] = rules
		}
	}
	return result
}

func filterIPLists(ipRuleMap model.IPRuleMap, lists []string) model.IPRuleMap {
	if len(lists) == 0 {
		return ipRuleMap
	}

	wanted := make(map[string]bool)
	for _, l := range lists {
		wanted[strings.ToLower(strings.TrimSpace(l))] = true
	}

	result := make(model.IPRuleMap)
	for code, cidrs := range ipRuleMap {
		if wanted[code] {
			result[code] = cidrs
		}
	}
	return result
}
