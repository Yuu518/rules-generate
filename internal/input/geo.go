package input

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/Yuu518/rules-generate/internal/model"
	"github.com/metacubex/mihomo/component/geodata/router"
	"google.golang.org/protobuf/proto"
)

func ParseDat(filePath string) (model.RuleMap, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	list := router.GeoSiteList{}
	if err = proto.Unmarshal(data, &list); err != nil {
		return nil, err
	}

	ruleMap := make(model.RuleMap)

	for _, entry := range list.Entry {
		code := strings.ToLower(entry.CountryCode)

		var rules []model.DomainRule

		for _, domain := range entry.Domain {
			rule := convertDomain(domain)
			rules = append(rules, rule)
		}

		ruleMap[code] = rules
	}

	return ruleMap, nil
}

func ParseGeoIPDat(filePath string) (model.IPRuleMap, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	list := router.GeoIPList{}
	if err = proto.Unmarshal(data, &list); err != nil {
		return nil, err
	}

	ipRuleMap := make(model.IPRuleMap)

	for _, entry := range list.Entry {
		code := strings.ToLower(entry.CountryCode)
		var cidrs []string
		for _, cidr := range entry.Cidr {
			cidrStr := fmt.Sprintf("%s/%d", net.IP(cidr.Ip).String(), cidr.Prefix)
			cidrs = append(cidrs, cidrStr)
		}
		ipRuleMap[code] = cidrs
	}

	return ipRuleMap, nil
}

func convertDomain(domain *router.Domain) model.DomainRule {
	rule := model.DomainRule{
		Value: domain.Value,
	}
	switch domain.Type {
	case router.Domain_Full:
		rule.Type = model.DomainFull
	case router.Domain_Domain:
		rule.Type = model.DomainSuffix
	case router.Domain_Regex:
		rule.Type = model.DomainRegexp
	case router.Domain_Plain:
		rule.Type = model.DomainKeyword
	}
	return rule
}
