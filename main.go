package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/Yuu518/rules-generate/internal/input"
	"github.com/Yuu518/rules-generate/internal/model"
	"github.com/Yuu518/rules-generate/internal/output"
	"github.com/Yuu518/rules-generate/internal/resolver"
	"github.com/spf13/cobra"
)

var (
	outputDir    string
	formatStr    string
	listsStr     string
	excludeAttrs string
	concurrency  int
)

var rootCmd = &cobra.Command{
	Use:   "rules-generate",
	Short: "Convert v2fly/domain-list-community rules to sing-box/Surge/Loon formats",
}

var domaindirCmd = &cobra.Command{
	Use:   "domain",
	Short: "Convert from local domain data directory",
	RunE:  runDomainDir,
}

var geositeCmd = &cobra.Command{
	Use:   "geosite",
	Short: "Convert from geosite.dat protobuf file",
	RunE:  runGeosite,
}

var geoipCmd = &cobra.Command{
	Use:   "geoip",
	Short: "Convert from geoip.dat protobuf file",
	RunE:  runGeoIP,
}

var ipdirCmd = &cobra.Command{
	Use:   "ip",
	Short: "Convert from local IP data directory",
	RunE:  runIPDir,
}

var (
	domainDirPath string
	geositeFile   string
	geoipFile     string
	ipDirPath     string
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&outputDir, "output", "o", "./output", "Output directory")
	rootCmd.PersistentFlags().StringVarP(&formatStr, "format", "f", "all", "Output formats: singbox,surge,loon,all")
	rootCmd.PersistentFlags().StringVarP(&listsStr, "lists", "l", "", "Lists to export (comma-separated, default: all)")
	rootCmd.PersistentFlags().StringVar(&excludeAttrs, "exclude-attrs", "cn@!cn@ads,geolocation-cn@!cn@ads,geolocation-!cn@cn@ads", "Exclude rules with attributes, e.g. cn@!cn@ads,geolocation-cn@!cn")
	rootCmd.PersistentFlags().IntVarP(&concurrency, "concurrency", "j", runtime.NumCPU(), "Concurrency level")

	domaindirCmd.Flags().StringVarP(&domainDirPath, "datapath", "d", "", "Path to domain data directory (required)")
	domaindirCmd.MarkFlagRequired("datapath")

	geositeCmd.Flags().StringVar(&geositeFile, "file", "", "Path to geosite.dat file (required)")
	geositeCmd.MarkFlagRequired("file")

	geoipCmd.Flags().StringVar(&geoipFile, "file", "", "Path to geoip.dat file (required)")
	geoipCmd.MarkFlagRequired("file")

	ipdirCmd.Flags().StringVarP(&ipDirPath, "datapath", "d", "", "Path to IP data directory (required)")
	ipdirCmd.MarkFlagRequired("datapath")

	rootCmd.AddCommand(domaindirCmd)
	rootCmd.AddCommand(geositeCmd)
	rootCmd.AddCommand(geoipCmd)
	rootCmd.AddCommand(ipdirCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runDomainDir(cmd *cobra.Command, args []string) error {
	fmt.Printf("Parsing data directory: %s\n", domainDirPath)

	lm, err := input.ParseDir(domainDirPath)
	if err != nil {
		return fmt.Errorf("parse directory: %w", err)
	}

	fmt.Printf("Parsed %d lists\n", len(lm))

	if err := resolver.Resolve(lm); err != nil {
		return fmt.Errorf("resolve: %w", err)
	}

	excludes := resolver.ParseExcludeAttrs(excludeAttrs)
	ruleMap := resolver.ToRuleMap(lm, excludes)

	return exportDomain(ruleMap)
}

func runGeosite(cmd *cobra.Command, args []string) error {
	fmt.Printf("Parsing geosite dat file: %s\n", geositeFile)

	ruleMap, err := input.ParseDat(geositeFile)
	if err != nil {
		return fmt.Errorf("parse dat: %w", err)
	}

	fmt.Printf("Parsed %d lists\n", len(ruleMap))

	return exportDomain(ruleMap)
}

func runGeoIP(cmd *cobra.Command, args []string) error {
	fmt.Printf("Parsing geoip dat file: %s\n", geoipFile)

	ipRuleMap, err := input.ParseGeoIPDat(geoipFile)
	if err != nil {
		return fmt.Errorf("parse geoip dat: %w", err)
	}

	fmt.Printf("Parsed %d IP lists\n", len(ipRuleMap))

	return exportIP(ipRuleMap)
}

func runIPDir(cmd *cobra.Command, args []string) error {
	fmt.Printf("Parsing IP data directory: %s\n", ipDirPath)

	ipRuleMap, err := input.ParseIPDir(ipDirPath)
	if err != nil {
		return fmt.Errorf("parse ip directory: %w", err)
	}

	fmt.Printf("Parsed %d IP lists\n", len(ipRuleMap))

	return exportIP(ipRuleMap)
}

func exportDomain(ruleMap model.RuleMap) error {
	lists := parseLists(listsStr)
	formats := parseFormats(formatStr)
	splitByFormat := len(formats) > 1

	fmt.Printf("Exporting %d rule sets to %s (formats: %s)\n", len(ruleMap), outputDir, strings.Join(formats, ", "))

	for _, format := range formats {
		var err error
		switch format {
		case "singbox":
			fmt.Println("Exporting sing-box rules...")
			err = output.ExportSingBox(ruleMap, outputDir, lists, concurrency, splitByFormat)
		case "surge":
			fmt.Println("Exporting Surge rules...")
			err = output.ExportSurge(ruleMap, outputDir, lists, concurrency, splitByFormat)
		case "loon":
			fmt.Println("Exporting Loon rules...")
			err = output.ExportLoon(ruleMap, outputDir, lists, concurrency, splitByFormat)
		}
		if err != nil {
			return err
		}
	}

	fmt.Println("Done!")
	return nil
}

func exportIP(ipRuleMap model.IPRuleMap) error {
	lists := parseLists(listsStr)
	formats := parseFormats(formatStr)
	splitByFormat := len(formats) > 1

	fmt.Printf("Exporting %d IP rule sets to %s (formats: %s)\n", len(ipRuleMap), outputDir, strings.Join(formats, ", "))

	for _, format := range formats {
		var err error
		switch format {
		case "singbox":
			fmt.Println("Exporting sing-box IP rules...")
			err = output.ExportSingBoxIP(ipRuleMap, outputDir, lists, concurrency, splitByFormat)
		case "surge":
			fmt.Println("Exporting Surge IP rules...")
			err = output.ExportSurgeIP(ipRuleMap, outputDir, lists, concurrency, splitByFormat)
		case "loon":
			fmt.Println("Exporting Loon IP rules...")
			err = output.ExportLoonIP(ipRuleMap, outputDir, lists, concurrency, splitByFormat)
		}
		if err != nil {
			return err
		}
	}

	fmt.Println("Done!")
	return nil
}

func parseLists(s string) []string {
	if s == "" {
		return nil
	}
	var lists []string
	for _, l := range strings.Split(s, ",") {
		l = strings.TrimSpace(l)
		if l != "" {
			lists = append(lists, l)
		}
	}
	return lists
}

func parseFormats(s string) []string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" || s == "all" {
		return []string{"singbox", "surge", "loon"}
	}
	var formats []string
	for _, f := range strings.Split(s, ",") {
		f = strings.TrimSpace(f)
		if f != "" {
			formats = append(formats, f)
		}
	}
	return formats
}
