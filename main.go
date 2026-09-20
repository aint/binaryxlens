package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aint/binaryxlens/internal"
	"github.com/aint/binaryxlens/internal/polygonscan"
)

// defaultExplorerAPIKey is the fallback when POLYGONSCAN_API_KEY and -api-key are empty.
// Prefer env/flag in shared repos so the key is not committed; rotate if this key leaks.
const defaultExplorerAPIKey = ""

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func main() {
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	apiKey := fs.String("api-key", getenv("POLYGONSCAN_API_KEY", defaultExplorerAPIKey), "Etherscan API v2 key (overrides POLYGONSCAN_API_KEY; default is built-in)")
	scanPause := fs.Duration("scan-pause", 400*time.Millisecond, "Extra pause between tokentx pages (free tier is often ~3 req/sec; client also spaces every call)")
	topHolders := fs.Int("top-holders", 25, "Show this many largest holders in report (0 = all)")
	_ = fs.Parse(os.Args[1:])

	client := polygonscan.NewClinet(*apiKey)
	projects := initAllProjects(client, *scanPause)
	for _, project := range projects {
		err := project.GenerateReport(*topHolders)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate %q report: %v\n", project.Name, err)
			os.Exit(1)
		}
	}
}

func initAllProjects(client *polygonscan.Client, scanPause time.Duration) []*internal.Project {
	var projects []*internal.Project
	for name, contracts := range internal.AllPropertyContracts {
		var properties []*internal.Property
		for _, contract := range contracts {
			property, err := internal.NewProperty(contract, client, scanPause)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create %q property: %v\n", name, err)
				continue
			}
			properties = append(properties, property)
		}
		project, err := internal.NewProject(name, properties)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create %q project: %v\n", name, err)
			continue
		}
		projects = append(projects, project)
	}
	return projects
}
