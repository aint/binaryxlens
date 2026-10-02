package internal

import (
	"maps"
	"math/big"
	"slices"
	"time"
)

const zeroAddr0x = "0x0000000000000000000000000000000000000000"

type Holder struct {
	Address       string
	Balance       *big.Int
	WeekDelta     *big.Int
	MonthDelta    *big.Int
	InitialBought *big.Int
	P2PBought     *big.Int
}

type ProjectHolder struct {
	Address          string
	PropertyBalances map[string]*big.Int
	TotalBalance     *big.Int
	WeekDelta        *big.Int
	MonthDelta       *big.Int
	InitialBought    *big.Int
	P2PBought        *big.Int
}

func (p *Property) buildHolders() {
	now := time.Now().UTC()
	weekAgo := now.AddDate(0, 0, -7)
	monthAgo := now.AddDate(0, 0, -30)

	holderMap := make(map[string]*Holder)
	for _, t := range p.transfers {
		v := t.Value
		inWeek := !t.Time.Before(weekAgo)
		inMonth := !t.Time.Before(monthAgo)

		if t.From != zeroAddr0x {
			updateHolder(holderMap, t.From, new(big.Int).Neg(v), inWeek, inMonth)
		}
		if t.To != zeroAddr0x {
			updateHolder(holderMap, t.To, v, inWeek, inMonth)
			to := holderMap[t.To]
			if p.isInitialSale(t.From) {
				to.InitialBought.Add(to.InitialBought, v)
			} else if p.isP2PTransfer(t.From, t.To) {
				to.P2PBought.Add(to.P2PBought, v)
			}
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
}

func updateHolder(m map[string]*Holder, addr string, v *big.Int, inWeek, inMonth bool) {
	h := m[addr]
	if h == nil {
		h = &Holder{
			Address:       addr,
			Balance:       big.NewInt(0),
			WeekDelta:     big.NewInt(0),
			MonthDelta:    big.NewInt(0),
			InitialBought: big.NewInt(0),
			P2PBought:     big.NewInt(0),
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
					InitialBought:    new(big.Int).Set(hol.InitialBought),
					P2PBought:        new(big.Int).Set(hol.P2PBought),
				}
				projectHolderMap[hol.Address] = ph
				continue
			}

			ph.PropertyBalances[property.Name] = new(big.Int).Set(hol.Balance)
			ph.TotalBalance.Add(ph.TotalBalance, hol.Balance)
			ph.WeekDelta.Add(ph.WeekDelta, hol.WeekDelta)
			ph.MonthDelta.Add(ph.MonthDelta, hol.MonthDelta)
			ph.InitialBought.Add(ph.InitialBought, hol.InitialBought)
			ph.P2PBought.Add(ph.P2PBought, hol.P2PBought)
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
			InitialBought: FormatBigInt(h.InitialBought, pr.Decimal),
			P2PBought:     FormatBigInt(h.P2PBought, pr.Decimal),
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
	{0.5, "shrimp"},
	{1, "crab"},
	{5, "fish"},
	{10, "dolphin"},
	{20, "shark"},
	{100, "whale"},
}

func holderTier(percent float64) string {
	for _, t := range holderTierThresholds {
		if percent <= t.max {
			return t.name
		}
	}
	return holderTierThresholds[len(holderTierThresholds)-1].name
}
