package internal

import (
	"bytes"
	_ "embed"
	"fmt"
	"html"
	"math/big"
	"os"
	"slices"
	"strings"

	"github.com/aint/binaryxlens/internal/polygonscan"
)

//go:embed holders.html
var holdersTemplateSrc []byte

var holdersRowsPlaceholder = []byte("__HOLDERS_ROWS__")

type holderProperty struct {
	Name    string
	Project string
}

type globalHolder struct {
	Address     string
	Properties  []holderProperty
	TotalUSDT   *big.Int
	OffPlatform bool
}

func WriteHolders(path string, projects []*Project) error {
	if !bytes.Contains(holdersTemplateSrc, holdersRowsPlaceholder) {
		return fmt.Errorf("holders template missing placeholder")
	}
	out := bytes.ReplaceAll(holdersTemplateSrc, holdersRowsPlaceholder, []byte(holderRowsHTML(globalHolders(projects))))
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return fmt.Errorf("write holders: %w", err)
	}
	fmt.Println("Holders page is ready at", path)
	return nil
}

// globalHolders merges project holders by address. Only properties the wallet
// still holds are listed. USDT is the sum of those projects.
func globalHolders(projects []*Project) []globalHolder {
	byAddress := make(map[string]*globalHolder)
	for _, pr := range projects {
		for _, h := range pr.Holders {
			if h.TotalBalance.Sign() <= 0 {
				continue
			}
			g := byAddress[h.Address]
			if g == nil {
				g = &globalHolder{Address: h.Address, TotalUSDT: big.NewInt(0)}
				byAddress[h.Address] = g
			}
			for name, bal := range h.PropertyBalances {
				if bal.Sign() <= 0 {
					continue
				}
				g.Properties = append(g.Properties, holderProperty{Name: name, Project: pr.Name})
			}
			g.TotalUSDT.Add(g.TotalUSDT, h.InitSaleUSDT)
			g.TotalUSDT.Add(g.TotalUSDT, h.P2PUSDT)
			g.OffPlatform = g.OffPlatform || h.OffPlatformP2P
		}
	}

	holders := make([]globalHolder, 0, len(byAddress))
	for _, g := range byAddress {
		slices.SortFunc(g.Properties, func(a, b holderProperty) int {
			if c := strings.Compare(a.Name, b.Name); c != 0 {
				return c
			}
			return strings.Compare(a.Project, b.Project)
		})
		holders = append(holders, *g)
	}
	slices.SortFunc(holders, func(a, b globalHolder) int {
		if c := b.TotalUSDT.Cmp(a.TotalUSDT); c != 0 {
			return c
		}
		return strings.Compare(a.Address, b.Address)
	})
	return holders
}

func holderRowsHTML(holders []globalHolder) string {
	var b strings.Builder
	for _, h := range holders {
		mark := ""
		if h.OffPlatform {
			mark = "*"
		}
		fmt.Fprintf(&b, `<tr>
            <td class="address">%s</td>
            <td class="properties">%s</td>
            <td class="num">%d</td>
            <td class="num"><span class="usdt">%s</span>%s</td>
          </tr>
`,
			html.EscapeString(h.Address),
			propertyLinksHTML(h.Properties),
			len(h.Properties),
			FormatBigInt(h.TotalUSDT, polygonscan.USDTDecimal),
			mark,
		)
	}
	return b.String()
}

func propertyLinksHTML(properties []holderProperty) string {
	var b strings.Builder
	for _, p := range properties {
		fmt.Fprintf(&b, "\n              <a href=\"reports/%s\">%s</a>", html.EscapeString(projectReportFile(p.Project)), html.EscapeString(p.Name))
	}
	b.WriteString("\n            ")
	return b.String()
}
