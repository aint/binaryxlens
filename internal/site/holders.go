package site

import (
	"bytes"
	_ "embed"
	"fmt"
	"html"
	"math/big"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/aint/binaryxlens/internal/binaryx"
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
	Address        string
	Properties     []holderProperty
	TotalUSDT      *big.Int
	MonthDeltaUSDT *big.Int
	OffPlatform    bool
}

func WriteHolders(path string, updated time.Time, projects []*binaryx.Project) error {
	stamp := updated.UTC().Format(reportsUpdatedLayout)
	if !bytes.Contains(holdersTemplateSrc, holdersRowsPlaceholder) || !bytes.Contains(holdersTemplateSrc, indexUpdatedPlaceholder) || !bytes.Contains(holdersTemplateSrc, i18nPlaceholder) {
		return fmt.Errorf("holders template missing placeholder")
	}
	out := bytes.ReplaceAll(withI18n(holdersTemplateSrc), indexUpdatedPlaceholder, []byte(stamp))
	out = bytes.ReplaceAll(out, holdersRowsPlaceholder, []byte(holderRowsHTML(globalHolders(projects))))
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return fmt.Errorf("write holders: %w", err)
	}
	fmt.Println("Holders page is ready at", path)
	return nil
}

// globalHolders merges project holders by address. Only properties the wallet
// still holds are listed. USDT is the sum of those projects.
func globalHolders(projects []*binaryx.Project) []globalHolder {
	byAddress := make(map[string]*globalHolder)
	for _, pr := range projects {
		for _, h := range pr.Holders {
			g := byAddress[h.Address]
			if g == nil {
				g = &globalHolder{Address: h.Address, TotalUSDT: big.NewInt(0), MonthDeltaUSDT: big.NewInt(0)}
				byAddress[h.Address] = g
			}
			g.MonthDeltaUSDT.Add(g.MonthDeltaUSDT, h.MonthDeltaUSDT)
			if h.TotalBalance.Sign() <= 0 {
				continue
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
	for address, g := range byAddress {
		if len(g.Properties) == 0 {
			// The wallet holds nothing anywhere, it is not a holder.
			delete(byAddress, address)
			continue
		}
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
            %s
          </tr>`,
			html.EscapeString(h.Address),
			propertyLinksHTML(h.Properties),
			len(h.Properties),
			binaryx.FormatBigInt(h.TotalUSDT, polygonscan.USDTDecimal),
			mark,
			holderMonthDeltaHTML(h.MonthDeltaUSDT),
		)
	}
	return b.String()
}

// TODO; review, remove, commit again!

// holderMonthDeltaHTML renders the last 30 days USDT change of a holder.
// The sign stays outside the .usdt span: the page script reformats .usdt
// numbers with toLocaleString, which would drop a leading plus.
func holderMonthDeltaHTML(delta *big.Int) string {
	if delta == nil || delta.Sign() == 0 {
		return `<td class="num trend flat">—</td>`
	}
	sign, class := "+", "up"
	if delta.Sign() < 0 {
		sign, class = "-", "down"
	}
	return fmt.Sprintf(`<td class="num trend %s">%s<span class="usdt">%s</span> USDT</td>`,
		class,
		sign,
		binaryx.FormatBigInt(new(big.Int).Abs(delta), polygonscan.USDTDecimal),
	)
}

func propertyLinksHTML(properties []holderProperty) string {
	var b strings.Builder
	for _, p := range properties {
		fmt.Fprintf(&b, "\n              <a href=\"reports/%s\">%s</a>", html.EscapeString(projectReportFile(p.Project)), html.EscapeString(p.Name))
	}
	b.WriteString("\n            ")
	return b.String()
}
