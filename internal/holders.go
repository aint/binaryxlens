package internal

import (
	"fmt"
	"maps"
	"math/big"
	"slices"
)

const zeroAddr0x = "0x0000000000000000000000000000000000000000"

type Holder struct {
	Address string
	Balance *big.Int
}

type ProjectHolder struct {
	Address          string
	PropertyBalances map[string]*big.Int
	TotalBalance     *big.Int
}

func (p *Property) buildHolders() error {
	holderMap := make(map[string]*big.Int)
	for _, tx := range p.txs {
		v, ok := new(big.Int).SetString(tx.Value, 10)
		if !ok {
			return fmt.Errorf("parse value %q", tx.Value)
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

	holders := make([]Holder, 0, len(holderMap))
	for _, address := range keys {
		if address == p.Address {
			// ignore unclaimed balance
			continue
		}
		holders = append(holders, Holder{Address: address, Balance: holderMap[address]})
	}

	p.Holders = holders

	return nil
}

func (pr *Project) buildHolders() {
	projectHolderMap := make(map[string]*ProjectHolder)

	for _, property := range pr.Properties {
		for _, hol := range property.Holders {
			ph := projectHolderMap[hol.Address]
			if ph == nil {
				ph = &ProjectHolder{
					Address:          hol.Address,
					PropertyBalances: map[string]*big.Int{property.Name: new(big.Int).Set(hol.Balance)},
					TotalBalance:     new(big.Int).Set(hol.Balance),
				}
				projectHolderMap[hol.Address] = ph
				continue
			}

			ph.PropertyBalances[property.Name] = new(big.Int).Set(hol.Balance)
			ph.TotalBalance.Add(ph.TotalBalance, hol.Balance)
		}
	}

	projectHolders := make([]ProjectHolder, 0, len(projectHolderMap))
	for _, ph := range projectHolderMap {
		projectHolders = append(projectHolders, *ph)
	}
	slices.SortFunc(projectHolders, func(a, b ProjectHolder) int {
		return b.TotalBalance.Cmp(a.TotalBalance)
	})

	pr.Holders = projectHolders
}

func (pr *Project) buildHoldersPayload() ([]projectHolderPayload, []tierStatPayload) {
	holders := make([]projectHolderPayload, 0, len(pr.Holders))
	pcts := make([]float64, 0, len(pr.Holders))
	for _, h := range pr.Holders {
		if h.TotalBalance.Sign() == 0 {
			continue
		}
		pct := PercentFloat(h.TotalBalance, pr.TotalSupplyRaw)
		holders = append(holders, projectHolderPayload{
			Address:       h.Address,
			PropertyNames: slices.Sorted(maps.Keys(h.PropertyBalances)),
			Balance:       FormatBigInt(h.TotalBalance, pr.Decimal),
			SupplyPct:     pct,
			Tier:          holderTier(pct),
		})
		pcts = append(pcts, pct)
	}
	return holders, buildTierStatsPayload(pcts)
}

func buildTierStatsPayload(pcts []float64) []tierStatPayload {
	total := len(pcts)
	if total == 0 {
		return nil
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

	out := make([]tierStatPayload, 0, len(holderTierThresholds))
	for _, t := range holderTierThresholds {
		s := stats[t.name]
		out = append(out, tierStatPayload{
			Name:       t.name,
			Count:      s.count,
			HoldersPct: float64(s.count) / float64(total) * 100,
			SupplyPct:  s.supplyPct,
		})
	}
	return out
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
