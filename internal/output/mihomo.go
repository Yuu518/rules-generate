package output

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	P "github.com/metacubex/mihomo/constant/provider"
	RP "github.com/metacubex/mihomo/rules/provider"
	"gopkg.in/yaml.v3"

	"github.com/Yuu518/rules-generate/internal/model"
)

func ExportMihomo(ruleMap model.RuleMap, outputDir string, lists []string, concurrency int, splitByFormat bool) error {
	dir := formatOutputDir(outputDir, "mihomo", splitByFormat)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	classicalDir := filepath.Join(dir, "classical")
	if err := os.MkdirAll(classicalDir, 0755); err != nil {
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

			if err := saveMihomoDomainRuleSet(rules, dir, classicalDir, code); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("%s: %w", code, err))
				mu.Unlock()
			}
		}(code, rules)
	}

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("mihomo export errors: %v", errors)
	}
	return nil
}

func ExportMihomoIP(ipRuleMap model.IPRuleMap, outputDir string, lists []string, concurrency int, splitByFormat bool) error {
	dir := formatOutputDir(outputDir, "mihomo", splitByFormat)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	classicalDir := filepath.Join(dir, "classical")
	if err := os.MkdirAll(classicalDir, 0755); err != nil {
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

			if err := saveMihomoIPRuleSet(cidrs, dir, classicalDir, code); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("%s: %w", code, err))
				mu.Unlock()
			}
		}(code, cidrs)
	}

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("mihomo ip export errors: %v", errors)
	}
	return nil
}

func saveMihomoDomainRuleSet(rules []model.DomainRule, dir, classicalDir, code string) error {
	var domainPayload []string
	var classicalPayload []string

	for _, rule := range rules {
		switch rule.Type {
		case model.DomainFull:
			domainPayload = append(domainPayload, rule.Value)
			classicalPayload = append(classicalPayload, "DOMAIN,"+rule.Value)
		case model.DomainSuffix:
			domainPayload = append(domainPayload, "+."+rule.Value)
			classicalPayload = append(classicalPayload, "DOMAIN-SUFFIX,"+rule.Value)
		case model.DomainKeyword:
			classicalPayload = append(classicalPayload, "DOMAIN-KEYWORD,"+rule.Value)
		case model.DomainRegexp:
			classicalPayload = append(classicalPayload, "DOMAIN-REGEX,"+rule.Value)
		}
	}

	if len(domainPayload) == 0 && len(classicalPayload) == 0 {
		return nil
	}

	if len(domainPayload) > 0 {
		if err := saveMihomoYAML(domainPayload, filepath.Join(dir, code+".yaml")); err != nil {
			return err
		}
		domainText := strings.Join(domainPayload, "\n")
		if err := os.WriteFile(filepath.Join(dir, code+".list"), []byte(domainText), 0644); err != nil {
			return err
		}
		if err := saveMihomoMRS([]byte(domainText), "domain", filepath.Join(dir, code+".mrs")); err != nil {
			return err
		}
	}

	if len(classicalPayload) > 0 {
		if err := saveMihomoYAML(classicalPayload, filepath.Join(classicalDir, code+".yaml")); err != nil {
			return err
		}
		classicalText := strings.Join(classicalPayload, "\n")
		if err := os.WriteFile(filepath.Join(classicalDir, code+".list"), []byte(classicalText), 0644); err != nil {
			return err
		}
	}

	return nil
}

func saveMihomoIPRuleSet(cidrs []string, dir, classicalDir, code string) error {
	var ipPayload []string
	var classicalPayload []string

	for _, cidr := range cidrs {
		ip, _, err := net.ParseCIDR(cidr)
		if err != nil {
			ip = net.ParseIP(cidr)
		}
		if ip == nil {
			continue
		}
		ipPayload = append(ipPayload, cidr)
		if ip.To4() != nil {
			classicalPayload = append(classicalPayload, "IP-CIDR,"+cidr)
		} else {
			classicalPayload = append(classicalPayload, "IP-CIDR6,"+cidr)
		}
	}

	if len(ipPayload) == 0 {
		return nil
	}

	if err := saveMihomoYAML(ipPayload, filepath.Join(dir, code+".yaml")); err != nil {
		return err
	}

	ipText := strings.Join(ipPayload, "\n")
	if err := os.WriteFile(filepath.Join(dir, code+".list"), []byte(ipText), 0644); err != nil {
		return err
	}

	if err := saveMihomoMRS([]byte(ipText), "ipcidr", filepath.Join(dir, code+".mrs")); err != nil {
		return err
	}

	if len(classicalPayload) > 0 {
		if err := saveMihomoYAML(classicalPayload, filepath.Join(classicalDir, code+".yaml")); err != nil {
			return err
		}
		classicalText := strings.Join(classicalPayload, "\n")
		if err := os.WriteFile(filepath.Join(classicalDir, code+".list"), []byte(classicalText), 0644); err != nil {
			return err
		}
	}

	return nil
}

func saveMihomoYAML(payload []string, outputPath string) error {
	data := map[string][]string{
		"payload": payload,
	}
	buf, err := yaml.Marshal(&data)
	if err != nil {
		return fmt.Errorf("yaml marshal: %w", err)
	}
	return os.WriteFile(outputPath, buf, 0644)
}

func saveMihomoMRS(buf []byte, behavior string, outputPath string) error {
	b, err := P.ParseBehavior(behavior)
	if err != nil {
		return err
	}
	f, err := P.ParseRuleFormat("text")
	if err != nil {
		return err
	}
	targetFile, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	err = RP.ConvertToMrs(buf, b, f, targetFile)
	closeErr := targetFile.Close()
	if err != nil {
		return err
	}
	return closeErr
}
