package internal

import (
	"bytes"
	_ "embed"
	"fmt"
	"html"
	"os"
	"strings"
)

//go:embed holders.html
var holdersTemplateSrc []byte

var holdersRowsPlaceholder = []byte("__HOLDERS_ROWS__")

type mockHolder struct {
	Address     string
	Properties  []string
	Total       string
	OffPlatform bool
}

func WriteHolders(path string) error {
	if !bytes.Contains(holdersTemplateSrc, holdersRowsPlaceholder) {
		return fmt.Errorf("holders template missing placeholder")
	}
	out := bytes.ReplaceAll(holdersTemplateSrc, holdersRowsPlaceholder, []byte(mockHolderRowsHTML()))
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return fmt.Errorf("write holders: %w", err)
	}
	fmt.Println("Holders page is ready at", path)
	return nil
}

func mockHolders() []mockHolder {
	return []mockHolder{
		{
			Address:    "0x8f3c91a2b7e64d0c15a8f6e2d49b7103c4aa91de",
			Properties: []string{"Dukley", "Bubbles Boutique Complex", "Kammara Loft", "Oasis Royal Collection"},
			Total:      "968550.50",
		},
		{
			Address:    "0x41ab07c9e2d58f10a6b33477c0e91d28f5c6aa10",
			Properties: []string{"Hayat Green Tower", "Ecoverse Suites"},
			Total:      "410000.00",
		},
		{
			Address:    "0xc0ee15d84a7093b217f66e90d12ab4487731c4e2",
			Properties: []string{"Aurora Villas (Bali Balance Ocean Villas)", "Roots Villas", "Cascade Villas"},
			Total:      "304010.25",
		},
		{
			Address:    "0x19d4aa80c3f1726e90b81a4475c0de44f8bb2201",
			Properties: []string{"Vesna Townhouse"},
			Total:      "198000.00",
		},
		{
			Address:     "0x77b21e04a9c6d58310f0aa22e6d91c48b30477a9",
			Properties:  []string{"Kammora Living", "Karra Loft"},
			Total:       "64000.00",
			OffPlatform: true,
		},
		{
			Address:    "0x02fa88c1d4e7095b6a3310c78e92d14f5bb00173",
			Properties: []string{"Tropical Loft Villas", "Bingin Magic Story Villas", "La Casa Española Villas"},
			Total:      "31050.75",
		},
		{
			Address:    "0xbb4091de77c0a218f55e04c3912aa80d16c7f304",
			Properties: []string{"Onyx Parq Resort 61"},
			Total:      "12480.00",
		},
		{
			Address:    "0x5e10c8a44b90d3172f66aa01c9e4d80b7721cc55",
			Properties: []string{"Cemagi Units", "Taryan Dragon Jungle View"},
			Total:      "5400.00",
		},
	}
}

func mockHolderRowsHTML() string {
	var b strings.Builder
	for _, h := range mockHolders() {
		total := h.Total
		if h.OffPlatform {
			total += "*"
		}
		fmt.Fprintf(&b, `<tr>
            <td class="address">%s</td>
            <td class="properties">%s</td>
            <td class="num">%d</td>
            <td class="num">%s</td>
          </tr>
`,
			html.EscapeString(h.Address),
			propertyLinksHTML(h.Properties),
			len(h.Properties),
			html.EscapeString(total),
		)
	}
	return b.String()
}

func propertyLinksHTML(names []string) string {
	var b strings.Builder
	for _, name := range names {
		fmt.Fprintf(&b, "\n              <a href=\"reports/%s\">%s</a>", html.EscapeString(projectReportFile(name)), html.EscapeString(name))
	}
	b.WriteString("\n            ")
	return b.String()
}
