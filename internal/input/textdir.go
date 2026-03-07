package input

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/Yuu518/rules-generate/internal/model"
)

func ParseDir(dataPath string) (model.ListInfoMap, error) {
	lm := make(model.ListInfoMap)
	err := filepath.Walk(dataPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		listInfo, err := ParseFile(path)
		if err != nil {
			return err
		}
		listName := model.FileName(strings.ToUpper(filepath.Base(path)))
		listInfo.Name = listName
		lm[listName] = listInfo
		return nil
	})
	if err != nil {
		return nil, err
	}
	return lm, nil
}

func ParseFile(path string) (*model.ListInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	list := model.NewListInfo()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if isEmpty(line) {
			continue
		}
		line = removeComment(line)
		if isEmpty(line) {
			continue
		}
		rule, err := parseRule(line, list)
		if err != nil {
			return nil, err
		}
		if rule == nil {
			continue
		}
		classifyRule(rule, list)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func parseRule(line string, list *model.ListInfo) (*model.DomainRule, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, errors.New("empty line")
	}

	if strings.HasPrefix(line, "include:") {
		parseInclusion(line, list)
		return nil, nil
	}

	parts := strings.Split(line, " ")
	ruleWithType := strings.TrimSpace(parts[0])
	if ruleWithType == "" {
		return nil, errors.New("empty rule")
	}

	rule := model.DomainRule{}
	if err := parseTypeRule(ruleWithType, &rule); err != nil {
		return nil, err
	}

	for _, attrString := range parts[1:] {
		attrString = strings.TrimSpace(attrString)
		if attrString == "" {
			continue
		}
		attr, err := parseAttribute(attrString)
		if err != nil {
			return nil, err
		}
		rule.Attributes = append(rule.Attributes, attr)
	}

	return &rule, nil
}

func parseInclusion(inclusion string, list *model.ListInfo) {
	inclusionVal := strings.TrimPrefix(strings.TrimSpace(inclusion), "include:")
	list.HasInclusion = true
	inclusionValSlice := strings.Split(inclusionVal, "@")
	filename := model.FileName(strings.ToUpper(strings.TrimSpace(inclusionValSlice[0])))

	if len(inclusionValSlice) == 1 {
		list.InclusionAttributeMap[filename] = append(list.InclusionAttributeMap[filename], model.AttrKey("@"))
	} else {
		for _, attr := range inclusionValSlice[1:] {
			attr = strings.TrimSpace(attr)
			attr = strings.ToLower(attr)
			if attr != "" {
				list.InclusionAttributeMap[filename] = append(list.InclusionAttributeMap[filename], model.AttrKey("@"+attr))
			}
		}
	}
}

func parseTypeRule(domain string, rule *model.DomainRule) error {
	kv := strings.SplitN(domain, ":", 2)
	switch len(kv) {
	case 1:
		rule.Type = model.DomainSuffix
		rule.Value = strings.ToLower(strings.TrimSpace(kv[0]))
	case 2:
		ruleType := strings.TrimSpace(kv[0])
		ruleVal := strings.TrimSpace(kv[1])
		rule.Value = strings.ToLower(ruleVal)
		switch strings.ToLower(ruleType) {
		case "full":
			rule.Type = model.DomainFull
		case "domain":
			rule.Type = model.DomainSuffix
		case "keyword":
			rule.Type = model.DomainKeyword
		case "regexp":
			rule.Type = model.DomainRegexp
			rule.Value = ruleVal
		default:
			return errors.New("unknown domain type: " + ruleType)
		}
	}
	return nil
}

func parseAttribute(attr string) (string, error) {
	if attr[0] != '@' {
		return "", errors.New("invalid attribute: " + attr)
	}
	return strings.ToLower(attr[1:]), nil
}

func classifyRule(rule *model.DomainRule, list *model.ListInfo) {
	if len(rule.Attributes) > 0 {
		list.AttributeRuleList = append(list.AttributeRuleList, *rule)
		var attrsString model.AttrKey
		for _, attr := range rule.Attributes {
			attrsString += model.AttrKey("@" + attr)
		}
		list.AttributeRuleMap[attrsString] = append(list.AttributeRuleMap[attrsString], *rule)
	} else {
		switch rule.Type {
		case model.DomainFull:
			list.FullList = append(list.FullList, *rule)
		case model.DomainSuffix:
			list.SuffixList = append(list.SuffixList, *rule)
		case model.DomainKeyword:
			list.KeywordList = append(list.KeywordList, *rule)
		case model.DomainRegexp:
			list.RegexpList = append(list.RegexpList, *rule)
		}
	}
}

func isEmpty(s string) bool {
	return len(strings.TrimSpace(s)) == 0
}

func removeComment(line string) string {
	idx := strings.Index(line, "#")
	if idx == -1 {
		return line
	}
	return strings.TrimSpace(line[:idx])
}
