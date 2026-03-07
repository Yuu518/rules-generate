package input

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/Yuu518/rules-generate/internal/model"
)

func ParseIPDir(dataPath string) (model.IPRuleMap, error) {
	ipRuleMap := make(model.IPRuleMap)
	err := filepath.Walk(dataPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		cidrs, err := parseIPFile(path)
		if err != nil {
			return err
		}
		code := strings.ToLower(filepath.Base(path))
		code = strings.TrimSuffix(code, filepath.Ext(code))
		ipRuleMap[code] = cidrs
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ipRuleMap, nil
}

func parseIPFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cidrs []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		cidrs = append(cidrs, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return cidrs, nil
}
