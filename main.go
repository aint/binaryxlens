package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aint/binaryxlens/internal/site"
	"github.com/aint/binaryxlens/internal/binaryx"
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
	_ = fs.Parse(os.Args[1:])

	client := polygonscan.NewClinet(*apiKey)
	projects := initAllProjects(client, *scanPause)
	for _, project := range projects {
		err := site.GenerateReport(project)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate %q report: %v\n", project.Name, err)
			os.Exit(1)
		}
	}
	stamp := time.Now().UTC()
	if err := site.WriteIndex("index.html", stamp, projects); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write index: %v\n", err)
		os.Exit(1)
	}
	if err := site.WriteHolders("holders.html", stamp, projects); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write holders: %v\n", err)
		os.Exit(1)
	}
}

func initAllProjects(client *polygonscan.Client, scanPause time.Duration) []*binaryx.Project {
	allP2PTrades, err := binaryx.FetchP2PTrades(client, scanPause)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to fetch allP2P trades: %v\n", err)
		os.Exit(1)
	}
	monthlyRentalProfit, err := binaryx.FetchRentalProfit(client, scanPause)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to fetch monthly rental profit: %v\n", err)
		os.Exit(1)
	}

	var projects []*binaryx.Project
	for name, properties := range binaryx.AllProperties {
		for _, property := range properties {
			err := property.Init(client, scanPause, allP2PTrades[property.Address], monthlyRentalProfit[property.Address])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create %q property: %v\n", name, err)
				continue
			}
		}
		project, err := binaryx.NewProject(name, properties)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create %q project: %v\n", name, err)
			continue
		}
		projects = append(projects, project)
	}
	return projects
}
