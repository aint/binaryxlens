package internal

import (
	"fmt"
	"maps"
	"math/big"
	"slices"
	"strconv"
	"time"
)

const zeroAddr0x = "0x0000000000000000000000000000000000000000"

type Holder struct {
	Address    string
	Balance    *big.Int
	WeekDelta  *big.Int
	MonthDelta *big.Int
}

type ProjectHolder struct {
	Address          string
	PropertyBalances map[string]*big.Int
	TotalBalance     *big.Int
	WeekDelta        *big.Int
	MonthDelta       *big.Int
}

func (p *Property) buildHolders() error {
	now := time.Now().UTC()
	weekAgo := now.AddDate(0, 0, -7).Unix()
	monthAgo := now.AddDate(0, 0, -30).Unix()

	holderMap := make(map[string]*Holder)
	for _, tx := range p.txs {
		v, ok := new(big.Int).SetString(tx.Value, 10)
		if !ok {
			return fmt.Errorf("parse value %q", tx.Value)
		}
		ts, err := strconv.ParseInt(tx.TimeStamp, 10, 64)
		if err != nil {
			return fmt.Errorf("parse timestamp %q: %w", tx.TimeStamp, err)
		}
		inWeek := ts >= weekAgo
		inMonth := ts >= monthAgo

		if tx.From != zeroAddr0x {
			updateHolder(holderMap, tx.From, new(big.Int).Neg(v), inWeek, inMonth)
		}
		if tx.To != zeroAddr0x {
			updateHolder(holderMap, tx.To, v, inWeek, inMonth)
		}
	}

	keys := slices.Collect(maps.Keys(holderMap))
	slices.SortFunc(keys, func(a, b string) int {
		return holderMap[b].Balance.Cmp(holderMap[a].Balance) // descending by balance
	})

	holders := make([]Holder, 0, len(holderMap))
	for _, address := range keys {
		if address == p.Address {
			// ignore unclaimed balance
			continue
		}
		holders = append(holders, *holderMap[address])
	}

	p.Holders = holders

	return nil
}

func updateHolder(m map[string]*Holder, addr string, v *big.Int, inWeek, inMonth bool) {
	h := m[addr]
	if h == nil {
		h = &Holder{
			Address:    addr,
			Balance:    big.NewInt(0),
			WeekDelta:  big.NewInt(0),
			MonthDelta: big.NewInt(0),
		}
		m[addr] = h
	}
	h.Balance.Add(h.Balance, v)
	if inWeek {
		h.WeekDelta.Add(h.WeekDelta, v)
	}
	if inMonth {
		h.MonthDelta.Add(h.MonthDelta, v)
	}
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
					WeekDelta:        new(big.Int).Set(hol.WeekDelta),
					MonthDelta:       new(big.Int).Set(hol.MonthDelta),
				}
				projectHolderMap[hol.Address] = ph
				continue
			}

			ph.PropertyBalances[property.Name] = new(big.Int).Set(hol.Balance)
			ph.TotalBalance.Add(ph.TotalBalance, hol.Balance)
			ph.WeekDelta.Add(ph.WeekDelta, hol.WeekDelta)
			ph.MonthDelta.Add(ph.MonthDelta, hol.MonthDelta)
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
			WeekDelta:     formatDelta(h.WeekDelta, pr.Decimal),
			MonthDelta:    formatDelta(h.MonthDelta, pr.Decimal),
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
