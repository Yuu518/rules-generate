package resolver

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/Yuu518/rules-generate/internal/model"
	"github.com/Yuu518/rules-generate/internal/trie"
)

func Resolve(lm model.ListInfoMap) error {
	inclusionLevel := make([]map[model.FileName]bool, 0, 20)
	okayList := make(map[model.FileName]bool)
	inclusionLevelAllLength, loopTimes := 0, 0

	for inclusionLevelAllLength < len(lm) {
		inclusionMap := make(map[model.FileName]bool)

		for _, listinfo := range lm {
			if okayList[listinfo.Name] {
				continue
			}
			if !listinfo.HasInclusion {
				inclusionMap[listinfo.Name] = true
				continue
			}
			allResolved := true
			for filename := range listinfo.InclusionAttributeMap {
				if _, exists := lm[filename]; exists && !okayList[filename] {
					allResolved = false
					break
				}
			}
			if allResolved {
				inclusionMap[listinfo.Name] = true
			}
		}

		for filename := range inclusionMap {
			okayList[filename] = true
		}

		inclusionLevel = append(inclusionLevel, inclusionMap)
		inclusionLevelAllLength += len(inclusionMap)
		loopTimes++

		if len(inclusionMap) == 0 && inclusionLevelAllLength < len(lm) {
			return fmt.Errorf("circular inclusion detected, resolved %d/%d lists", inclusionLevelAllLength, len(lm))
		}
	}

	for idx, inclusionMap := range inclusionLevel {
		fmt.Printf("Level %d: %d lists\n", idx+1, len(inclusionMap))
		for filename := range inclusionMap {
			if err := flatten(lm[filename], &lm); err != nil {
				return err
			}
		}
	}

	return nil
}

func flatten(l *model.ListInfo, lm *model.ListInfoMap) error {
	if l.HasInclusion {
		for filename, attrs := range l.InclusionAttributeMap {
			for _, attrWanted := range attrs {
				includedList := (*lm)[filename]
				if includedList == nil {
					continue
				}
				switch string(attrWanted) {
				case "@":
					l.FullList = append(l.FullList, includedList.FullList...)
					l.SuffixList = append(l.SuffixList, includedList.SuffixList...)
					l.KeywordList = append(l.KeywordList, includedList.KeywordList...)
					l.RegexpList = append(l.RegexpList, includedList.RegexpList...)
					l.AttributeRuleList = append(l.AttributeRuleList, includedList.AttributeRuleList...)
					for attr, domainList := range includedList.AttributeRuleMap {
						l.AttributeRuleMap[attr] = append(l.AttributeRuleMap[attr], domainList...)
					}
				default:
					for attr, domainList := range includedList.AttributeRuleMap {
						if strings.Contains(string(attr)+"@", string(attrWanted)+"@") {
							l.AttributeRuleMap[attr] = append(l.AttributeRuleMap[attr], domainList...)
							l.AttributeRuleList = append(l.AttributeRuleList, domainList...)
						}
					}
				}
			}
		}
	}

	sort.Slice(l.SuffixList, func(i, j int) bool {
		return len(strings.Split(l.SuffixList[i].Value, ".")) < len(strings.Split(l.SuffixList[j].Value, "."))
	})

	t := trie.NewDomainTrie()
	for _, domain := range l.SuffixList {
		success, err := t.Insert(domain.Value)
		if err != nil {
			return err
		}
		if success {
			l.SuffixUniqueList = append(l.SuffixUniqueList, domain)
		}
	}

	return nil
}

func ToRuleMap(lm model.ListInfoMap, excludeAttrs map[model.FileName]map[model.AttrKey]bool) model.RuleMap {
	ruleMap := make(model.RuleMap)

	for _, listInfo := range lm {
		code := strings.ToLower(string(listInfo.Name))

		var rules []model.DomainRule
		rules = append(rules, listInfo.FullList...)
		rules = append(rules, listInfo.SuffixUniqueList...)
		rules = append(rules, listInfo.RegexpList...)
		for _, keywordRule := range listInfo.KeywordList {
			if strings.TrimSpace(keywordRule.Value) != "" {
				rules = append(rules, keywordRule)
			}
		}

		excludes := excludeAttrs[listInfo.Name]
		for _, rule := range listInfo.AttributeRuleList {
			if excludes != nil {
				shouldExclude := false
				for _, attr := range rule.Attributes {
					if excludes[model.AttrKey(attr)] {
						shouldExclude = true
						break
					}
				}
				if shouldExclude {
					continue
				}
			}
			rules = append(rules, rule)
		}

		ruleMap[code] = rules
	}

	return ruleMap
}

func FilterTags(data model.RuleMap) {
	var codeList []string
	for code := range data {
		codeList = append(codeList, code)
	}

	type filteredPair struct {
		code    string
		badCode string
	}

	var badCodeList []filteredPair
	var filteredCodes []string

	for _, code := range codeList {
		codeParts := strings.Split(code, "@")
		if len(codeParts) != 2 {
			continue
		}
		leftParts := strings.Split(codeParts[0], "-")
		var lastName string
		if len(leftParts) > 1 {
			lastName = leftParts[len(leftParts)-1]
		}
		if lastName == "" {
			lastName = codeParts[0]
		}
		if lastName == codeParts[1] {
			delete(data, code)
			filteredCodes = append(filteredCodes, code)
			continue
		}
		if "!"+lastName == codeParts[1] || lastName == "!"+codeParts[1] {
			badCodeList = append(badCodeList, filteredPair{
				code:    codeParts[0],
				badCode: code,
			})
		}
	}

	var mergedCodes []string
	for _, it := range badCodeList {
		badList := data[it.badCode]
		if badList == nil {
			continue
		}
		delete(data, it.badCode)

		type ruleKey struct {
			Type  model.DomainType
			Value string
		}
		badSet := make(map[ruleKey]bool)
		for _, item := range badList {
			badSet[ruleKey{item.Type, item.Value}] = true
		}
		var newList []model.DomainRule
		for _, item := range data[it.code] {
			if !badSet[ruleKey{item.Type, item.Value}] {
				newList = append(newList, item)
			}
		}
		data[it.code] = newList
		mergedCodes = append(mergedCodes, it.badCode)
	}

	if len(filteredCodes) > 0 {
		sort.Strings(filteredCodes)
		fmt.Fprintf(os.Stderr, "filtered: %s\n", strings.Join(filteredCodes, ", "))
	}
	if len(mergedCodes) > 0 {
		sort.Strings(mergedCodes)
		fmt.Fprintf(os.Stderr, "merged: %s\n", strings.Join(mergedCodes, ", "))
	}
}

func MergeTags(data model.RuleMap) {
	var codeList []string
	for code := range data {
		codeList = append(codeList, code)
	}

	var cnCodeList []string

	for _, code := range codeList {
		codeParts := strings.Split(code, "@")
		if len(codeParts) != 2 {
			continue
		}
		if codeParts[1] != "cn" {
			continue
		}
		if !strings.HasPrefix(codeParts[0], "category-") {
			continue
		}
		if strings.HasSuffix(codeParts[0], "-cn") || strings.HasSuffix(codeParts[0], "-!cn") {
			continue
		}
		cnCodeList = append(cnCodeList, code)
	}

	for _, code := range codeList {
		if !strings.HasPrefix(code, "category-") {
			continue
		}
		if !strings.HasSuffix(code, "-cn") {
			continue
		}
		if strings.Contains(code, "@") {
			continue
		}
		cnCodeList = append(cnCodeList, code)
	}

	if len(cnCodeList) == 0 {
		return
	}

	type ruleKey struct {
		Type  model.DomainType
		Value string
	}
	ruleSet := make(map[ruleKey]model.DomainRule)
	for _, item := range data["geolocation-cn"] {
		ruleSet[ruleKey{item.Type, item.Value}] = item
	}
	for _, code := range cnCodeList {
		for _, item := range data[code] {
			ruleSet[ruleKey{item.Type, item.Value}] = item
		}
	}

	newList := make([]model.DomainRule, 0, len(ruleSet))
	for _, item := range ruleSet {
		newList = append(newList, item)
	}
	data["geolocation-cn"] = newList

	cnList := make([]model.DomainRule, len(newList), len(newList)+1)
	copy(cnList, newList)
	cnList = append(cnList, model.DomainRule{
		Type:  model.DomainSuffix,
		Value: "cn",
	})
	data["cn"] = cnList

	fmt.Printf("merged cn categories: %s\n", strings.Join(cnCodeList, ", "))
}

func ParseExcludeAttrs(excludeAttrsStr string) map[model.FileName]map[model.AttrKey]bool {
	result := make(map[model.FileName]map[model.AttrKey]bool)
	if excludeAttrsStr == "" {
		return result
	}

	entries := strings.Split(excludeAttrsStr, ",")
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.Split(entry, "@")
		filename := model.FileName(strings.ToUpper(strings.TrimSpace(parts[0])))
		result[filename] = make(map[model.AttrKey]bool)
		for _, attr := range parts[1:] {
			attr = strings.TrimSpace(attr)
			if attr != "" {
				result[filename][model.AttrKey(attr)] = true
			}
		}
	}
	return result
}

func SplitRuleMapByAttrs(data model.RuleMap, attrs []string) {
	if len(attrs) == 0 {
		return
	}

	attrSet := make(map[string]bool, len(attrs))
	splitAll := false
	for _, attr := range attrs {
		attr = strings.ToLower(strings.TrimSpace(attr))
		if attr == "" {
			continue
		}
		if attr == "all" {
			splitAll = true
			continue
		}
		attrSet[attr] = true
	}
	if len(attrSet) == 0 && !splitAll {
		return
	}

	var codeList []string
	for code := range data {
		codeList = append(codeList, code)
	}

	type ruleKey struct {
		Type  model.DomainType
		Value string
	}

	for _, code := range codeList {
		if strings.Contains(code, "@") {
			continue
		}

		baseRules := data[code]
		if len(baseRules) == 0 {
			continue
		}

		splitMap := make(map[string][]model.DomainRule)
		for _, rule := range baseRules {
			if len(rule.Attributes) == 0 {
				continue
			}
			for _, attr := range rule.Attributes {
				attr = strings.ToLower(strings.TrimSpace(attr))
				if attr == "" {
					continue
				}
				if !splitAll && !attrSet[attr] {
					continue
				}
				targetCode := code + "@" + attr
				splitMap[targetCode] = append(splitMap[targetCode], rule)
			}
		}

		for targetCode, rules := range splitMap {
			merged := append(data[targetCode], rules...)
			seen := make(map[ruleKey]bool, len(merged))
			unique := make([]model.DomainRule, 0, len(merged))
			for _, rule := range merged {
				key := ruleKey{Type: rule.Type, Value: rule.Value}
				if seen[key] {
					continue
				}
				seen[key] = true
				unique = append(unique, rule)
			}
			data[targetCode] = unique
		}
	}
}
