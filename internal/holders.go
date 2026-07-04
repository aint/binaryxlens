package internal

import (
	"fmt"
	"maps"
	"math/big"
	"slices"
	"strings"
)

const zeroAddr0x = "0x0000000000000000000000000000000000000000"

type Holder struct {
	Address string
	Balance *big.Int
}

type ProjectHolder struct {
	Address       string
	TokenBalances map[string]*big.Int
	TotalBalance  *big.Int
}

type Holders []Holder

func (t Token) getHolders() (Holders, error) {
	holderMap := make(map[string]*big.Int)
	for _, tx := range t.Txs {
		v, ok := new(big.Int).SetString(tx.Value, 10)
		if !ok {
			return nil, fmt.Errorf("parse value %q", tx.Value)
		}
		if tx.From != zeroAddr0x {
			cur := holderMap[tx.From]
			if cur == nil {
				cur = big.NewInt(0)
			}
			holderMap[tx.From] = new(big.Int).Sub(cur, v)
		}
		if tx.To != zeroAddr0x {
			cur := holderMap[tx.To]
			if cur == nil {
				cur = big.NewInt(0)
			}
			holderMap[tx.To] = new(big.Int).Add(cur, v)
		}
	}

	keys := slices.Collect(maps.Keys(holderMap))
	slices.SortFunc(keys, func(a, b string) int {
		return holderMap[b].Cmp(holderMap[a]) // descending by balance
	})

	holders := make(Holders, 0, len(holderMap))
	for _, address := range keys {
		if address == t.Address {
			// ignore token's balance
			continue
		}
		holders = append(holders, Holder{Address: address, Balance: holderMap[address]})
	}
	return holders, nil
}

func (t Token) PrintHolders(top int) error {
	idx := min(len(t.Holders), top)

	fmt.Printf("\nHolders: showing %d of %d\n", len(t.Holders[:idx]), len(t.Holders))
	fmt.Printf("%4s %-42s %12s %12s %-12s\n", "#", "address", "balance", "% supply", "tier")
	for i, h := range t.Holders[:idx] {
		if h.Balance.Sign() == 0 {
			continue
		}
		pct := PercentFloat(h.Balance, t.TotalSupplyRaw)
		fmt.Printf(
			"%4d %-42s %12s %11.2f%% %-12s\n",
			i+1,
			h.Address,
			FormatBigInt(h.Balance, t.Decimal),
			pct,
			holderTier(pct),
		)
	}

	pcts := make([]float64, 0, len(t.Holders))
	for _, h := range t.Holders {
		if h.Balance.Sign() == 0 {
			continue
		}
		pcts = append(pcts, PercentFloat(h.Balance, t.TotalSupplyRaw))
	}

	printHolderTierStats(pcts)
	return nil
}

func (p Project) getHolders() []ProjectHolder {
	projectHolderMap := make(map[string]*ProjectHolder)

	for _, token := range p.Tokens {
		for _, th := range token.Holders {
			ph := projectHolderMap[th.Address]
			if ph == nil {
				ph = &ProjectHolder{
					Address: th.Address,
					TokenBalances: map[string]*big.Int{token.Name: new(big.Int).Set(th.Balance)},
					TotalBalance: new(big.Int).Set(th.Balance),
				}
				projectHolderMap[th.Address] = ph
				continue
			}

			ph.TokenBalances[token.Name] = new(big.Int).Set(th.Balance)
			ph.TotalBalance.Add(ph.TotalBalance, th.Balance)
		}
	}

	projectHolders := make([]ProjectHolder, 0, len(projectHolderMap))
	for _, ph := range projectHolderMap {
		projectHolders = append(projectHolders, *ph)
	}
	slices.SortFunc(projectHolders, func(a, b ProjectHolder) int {
		return b.TotalBalance.Cmp(a.TotalBalance)
	})
	return projectHolders
}

func (p Project) PrintHolders(top int) error {
	idx := min(len(p.Holders), top)

	fmt.Printf("\nHolders: showing %d of %d\n", len(p.Holders[:idx]), len(p.Holders))
	fmt.Printf("%4s %-42s %-96s %12s %12s %-12s\n", "#", "address", "token names", "balance", "% supply", "tier")
	for i, h := range p.Holders[:idx] {
		names := slices.Sorted(maps.Keys(h.TokenBalances))
		pct := PercentFloat(h.TotalBalance, p.TotalSupplyRaw)
		fmt.Printf(
			"%4d %-42s %-96s %12s %11.2f%% %-12s\n",
			i+1,
			h.Address,
			strings.Join(names, ", "),
			FormatBigInt(h.TotalBalance, p.Decimal),
			pct,
			holderTier(pct),
		)
	}

	pcts := make([]float64, 0, len(p.Holders))
	for _, h := range p.Holders {
		if h.TotalBalance.Sign() == 0 {
			continue
		}
		pcts = append(pcts, PercentFloat(h.TotalBalance, p.TotalSupplyRaw))
	}

	printHolderTierStats(pcts)
	return nil
}

func printHolderTierStats(pcts []float64) {
	total := len(pcts)
	if total == 0 {
		return
	}

	type tierStat struct {
		count     int
		supplyPct float64
	}
	stats := make(map[string]tierStat)
	for _, pct := range pcts {
		tier := holderTier(pct)
		s := stats[tier]
		s.count++
		s.supplyPct += pct
		stats[tier] = s
	}

	fmt.Printf("\nTier distribution (%d holders):\n", total)
	fmt.Printf("%-12s %6s %8s %10s\n", "tier", "count", "% holders", "% supply")
	for _, t := range holderTierThresholds {
		s := stats[t.name]
		fmt.Printf(
			"%-12s %6d %7.1f%% %9.2f%%\n",
			t.name,
			s.count,
			float64(s.count)/float64(total)*100,
			s.supplyPct,
		)
	}
}

var holderTierThresholds = []struct {
	max  float64
	name string
}{
	{0.5, "🦐 Shrimp"},
	{1, "🦀 Crab"},
	{5, "🐟 Fish"},
	{10, "🐬 Dolphin"},
	{20, "🦈 Shark"},
	{100, "🐋 Whale"},
}

func holderTier(percent float64) string {
	for _, t := range holderTierThresholds {
		if percent <= t.max {
			return t.name
		}
	}
	return holderTierThresholds[len(holderTierThresholds)-1].name
}
