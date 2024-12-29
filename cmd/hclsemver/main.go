package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/david1155/hclsemver/internal/terraform"
	"github.com/david1155/hclsemver/pkg/config"
	"github.com/david1155/hclsemver/pkg/version"
)

func processConfig(configFile string, workDir string, dryRun bool) error {
	// Read and parse config
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	// Get all tiers from config
	configTiers := config.GetTiersFromConfig(cfg)

	// Process each module
	for _, module := range cfg.Modules {
		for tier, _ := range module.Versions {
			// Skip the wildcard tier as it's only used for inheritance
			if tier == "*" {
				continue
			}

			// Get effective version config for this tier
			versionConfig, err := config.GetEffectiveVersionConfig(module, tier)
			if err != nil {
				log.Printf("Error getting version config for module '%s' tier '%s': %v", module.Source, tier, err)
				continue
			}

			// Get effective strategy
			strategy := config.GetEffectiveStrategy(module, tier)

			// Get effective force setting
			force := config.GetEffectiveForce(module, tier)

			// Parse the version/range
			newIsVer, newVer, newConstr, err := version.ParseVersionOrRange(versionConfig.Version)
			if err != nil {
				log.Printf("Error parsing version '%s' for module '%s': %v", versionConfig.Version, module.Source, err)
				continue
			}

			rootDir := filepath.Join(workDir, tier)
			if err := terraform.ScanAndUpdateModules(rootDir, module.Source, newIsVer, newVer, newConstr, versionConfig.Version, configTiers, strategy, dryRun, force); err != nil {
				log.Printf("Error processing module '%s' in tier '%s': %v", module.Source, tier, err)
				continue
			}

			log.Printf("Successfully processed module '%s' in tier '%s'", module.Source, tier)
		}
	}
	return nil
}

func mainWithFlags(args []string, workDir string) error {
	// Create a new flag set
	flags := flag.NewFlagSet("hclsemver", flag.ContinueOnError)

	// Set custom usage message
	flags.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: hclsemver [options]\n\n")
		fmt.Fprintf(os.Stderr, "A tool for managing semantic versioning in Terraform HCL files.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flags.PrintDefaults()
	}

	// Define flags
	configFile := flags.String("config", "", "Path to config file (JSON or YAML)")
	dir := flags.String("dir", "/work", "Directory to scan for Terraform files")
	dryRun := flags.Bool("dry-run", false, "Preview changes without modifying files")
	help := flags.Bool("help", false, "Display help information")

	// Parse flags
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}

	// Show help if requested
	if *help {
		flags.Usage()
		return nil
	}

	if *configFile == "" {
		flags.Usage()
		return fmt.Errorf("config file is required: -config path/to/config.yaml")
	}

	return processConfig(*configFile, *dir, *dryRun)
}

func main() {
	if err := mainWithFlags(os.Args[1:], "/work"); err != nil {
		log.Fatal(err)
	}
}
