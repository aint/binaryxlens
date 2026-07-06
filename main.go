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
	project, err := internal.NewProject("La Casa Española Villas", getTokens(client, *scanPause))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create project: %v\n", err)
		os.Exit(1)
	}
	err = project.GenerateReport(*topHolders)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate project report: %v\n", err)
		os.Exit(1)
	}

}

func getTokens(client *polygonscan.Client, scanPause time.Duration) []internal.Token {
	token4, err := internal.NewToken(internal.LaCasaEspanolaVilla4, client, scanPause)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create token: %v\n", err)
		os.Exit(1)
	}
	token6, err := internal.NewToken(internal.LaCasaEspanolaVilla6, client, scanPause)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create token: %v\n", err)
		os.Exit(1)
	}
	token8, err := internal.NewToken(internal.LaCasaEspanolaVilla8, client, scanPause)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create token: %v\n", err)
		os.Exit(1)
	}
	token9, err := internal.NewToken(internal.LaCasaEspanolaVilla9, client, scanPause)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create token: %v\n", err)
		os.Exit(1)
	}
	return []internal.Token{token4, token6, token8, token9}
}
