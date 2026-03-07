package output

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Yuu518/rules-generate/internal/model"
)

func ExportSurge(ruleMap model.RuleMap, outputDir string, lists []string, concurrency int, splitByFormat bool) error {
	return exportTextRuleSet(ruleMap, formatOutputDir(outputDir, "surge", splitByFormat), lists, concurrency)
}

func ExportLoon(ruleMap model.RuleMap, outputDir string, lists []string, concurrency int, splitByFormat bool) error {
	return exportTextRuleSet(ruleMap, formatOutputDir(outputDir, "loon", splitByFormat), lists, concurrency)
}

func exportTextRuleSet(ruleMap model.RuleMap, dir string, lists []string, concurrency int) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	targets := filterLists(ruleMap, lists)
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	for code, rules := range targets {
		wg.Add(1)
		go func(code string, rules []model.DomainRule) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := saveTextRuleSet(rules, filepath.Join(dir, code+".list")); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("%s: %w", code, err))
				mu.Unlock()
			}
		}(code, rules)
	}

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("text export errors: %v", errors)
	}
	return nil
}

func ExportSurgeIP(ipRuleMap model.IPRuleMap, outputDir string, lists []string, concurrency int, splitByFormat bool) error {
	return exportTextIPRuleSet(ipRuleMap, formatOutputDir(outputDir, "surge", splitByFormat), lists, concurrency)
}

func ExportLoonIP(ipRuleMap model.IPRuleMap, outputDir string, lists []string, concurrency int, splitByFormat bool) error {
	return exportTextIPRuleSet(ipRuleMap, formatOutputDir(outputDir, "loon", splitByFormat), lists, concurrency)
}

func exportTextIPRuleSet(ipRuleMap model.IPRuleMap, dir string, lists []string, concurrency int) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	targets := filterIPLists(ipRuleMap, lists)
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	for code, cidrs := range targets {
		wg.Add(1)
		go func(code string, cidrs []string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := saveTextIPRuleSet(cidrs, filepath.Join(dir, code+".list")); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("%s: %w", code, err))
				mu.Unlock()
			}
		}(code, cidrs)
	}

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("text ip export errors: %v", errors)
	}
	return nil
}

func saveTextIPRuleSet(cidrs []string, outputPath string) error {
	var lines []string

	for _, cidr := range cidrs {
		ip, _, err := net.ParseCIDR(cidr)
		if err != nil {
			ip = net.ParseIP(cidr)
		}
		if ip == nil {
			continue
		}
		if ip.To4() != nil {
			lines = append(lines, "IP-CIDR,"+cidr)
		} else {
			lines = append(lines, "IP-CIDR6,"+cidr)
		}
	}

	output := strings.Join(lines, "\n")
	if output != "" {
		output += "\n"
	}
	return os.WriteFile(outputPath, []byte(output), 0644)
}

func saveTextRuleSet(rules []model.DomainRule, outputPath string) error {
	var lines []string

	for _, rule := range rules {
		switch rule.Type {
		case model.DomainFull:
			lines = append(lines, "DOMAIN,"+rule.Value)
		case model.DomainSuffix:
			lines = append(lines, "DOMAIN-SUFFIX,"+rule.Value)
		case model.DomainKeyword:
			lines = append(lines, "DOMAIN-KEYWORD,"+rule.Value)
		}
	}

	output := strings.Join(lines, "\n")
	if output != "" {
		output += "\n"
	}
	return os.WriteFile(outputPath, []byte(output), 0644)
}
