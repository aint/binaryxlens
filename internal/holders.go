package internal

import (
	"maps"
	"math/big"
	"slices"
	"time"

	"github.com/aint/binaryxlens/internal/polygonscan"
)

const zeroAddr0x = "0x0000000000000000000000000000000000000000"

type Holder struct {
	Address          string
	Balance          *big.Int
	WeekDeltaTokens  *big.Int
	MonthDeltaTokens *big.Int
	WeekDeltaUSDT    *big.Int
	MonthDeltaUSDT   *big.Int
	InitialBought    *big.Int
	P2PBought        *big.Int
	InitSaleUSDT     *big.Int
	P2PUSDT          *big.Int
	OffPlatformP2P   bool
	Entered          time.Time
	Exited           time.Time
}

type ProjectHolder struct {
	Address          string
	PropertyBalances map[string]*big.Int
	TotalBalance     *big.Int
	WeekDeltaTokens  *big.Int
	MonthDeltaTokens *big.Int
	WeekDeltaUSDT    *big.Int
	MonthDeltaUSDT   *big.Int
	InitialBought    *big.Int
	P2PBought        *big.Int
	InitSaleUSDT     *big.Int
	P2PUSDT          *big.Int
	OffPlatformP2P   bool
	Entered          time.Time
	Exited           time.Time
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
			var paid *big.Int
			if p.isP2PTransfer(t.From, t.To) {
				paid = negBigInt(t.USDT)
			}
			updateHolder(holderMap, t.From, new(big.Int).Neg(v), paid, inWeek, inMonth)
			from := holderMap[t.From]
			if from.Balance.Sign() == 0 {
				from.Exited = t.Time
			}
		}
		if t.To != zeroAddr0x {
			to := holderMap[t.To]
			opening := to == nil || to.Balance.Sign() == 0
			updateHolder(holderMap, t.To, v, t.USDT, inWeek, inMonth)
			to = holderMap[t.To]
			if opening {
				to.Entered = t.Time
				to.Exited = time.Time{}
			}
			if p.isInitialSale(t.From) {
				to.InitialBought.Add(to.InitialBought, v)
				if t.USDT != nil {
					to.InitSaleUSDT.Add(to.InitSaleUSDT, t.USDT)
				}
			} else if p.isP2PTransfer(t.From, t.To) {
				to.P2PBought.Add(to.P2PBought, v)
				if t.USDT != nil {
					to.P2PUSDT.Add(to.P2PUSDT, t.USDT)
				}
				// Wallet-to-wallet with no platform payment.
				if t.USDT == nil || t.USDT.Sign() == 0 {
					to.OffPlatformP2P = true
				}
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
		h := holderMap[address]
		if h.Balance.Sign() == 0 && h.MonthDeltaTokens.Sign() == 0 && h.MonthDeltaUSDT.Sign() == 0 {
			continue
		}
		holders = append(holders, *h)
	}

	p.Holders = holders
}

func updateHolder(m map[string]*Holder, addr string, tokens, usdt *big.Int, inWeek, inMonth bool) {
	h := m[addr]
	if h == nil {
		h = &Holder{
			Address:          addr,
			Balance:          big.NewInt(0),
			WeekDeltaTokens:  big.NewInt(0),
			MonthDeltaTokens: big.NewInt(0),
			WeekDeltaUSDT:    big.NewInt(0),
			MonthDeltaUSDT:   big.NewInt(0),
			InitialBought:    big.NewInt(0),
			P2PBought:        big.NewInt(0),
			InitSaleUSDT:     big.NewInt(0),
			P2PUSDT:          big.NewInt(0),
		}
		m[addr] = h
	}
	h.Balance.Add(h.Balance, tokens)
	if inWeek {
		h.WeekDeltaTokens.Add(h.WeekDeltaTokens, tokens)
		if usdt != nil {
			h.WeekDeltaUSDT.Add(h.WeekDeltaUSDT, usdt)
		}
	}
	if inMonth {
		h.MonthDeltaTokens.Add(h.MonthDeltaTokens, tokens)
		if usdt != nil {
			h.MonthDeltaUSDT.Add(h.MonthDeltaUSDT, usdt)
		}
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
					WeekDeltaTokens:  new(big.Int).Set(hol.WeekDeltaTokens),
					MonthDeltaTokens: new(big.Int).Set(hol.MonthDeltaTokens),
					WeekDeltaUSDT:    new(big.Int).Set(hol.WeekDeltaUSDT),
					MonthDeltaUSDT:   new(big.Int).Set(hol.MonthDeltaUSDT),
					InitialBought:    new(big.Int).Set(hol.InitialBought),
					P2PBought:        new(big.Int).Set(hol.P2PBought),
					InitSaleUSDT:     new(big.Int).Set(hol.InitSaleUSDT),
					P2PUSDT:          new(big.Int).Set(hol.P2PUSDT),
					OffPlatformP2P:   hol.OffPlatformP2P,
					Entered:          hol.Entered,
					Exited:           hol.Exited,
				}
				projectHolderMap[hol.Address] = ph
				continue
			}

			ph.PropertyBalances[property.Name] = new(big.Int).Set(hol.Balance)
			ph.TotalBalance.Add(ph.TotalBalance, hol.Balance)
			ph.WeekDeltaTokens.Add(ph.WeekDeltaTokens, hol.WeekDeltaTokens)
			ph.MonthDeltaTokens.Add(ph.MonthDeltaTokens, hol.MonthDeltaTokens)
			ph.WeekDeltaUSDT.Add(ph.WeekDeltaUSDT, hol.WeekDeltaUSDT)
			ph.MonthDeltaUSDT.Add(ph.MonthDeltaUSDT, hol.MonthDeltaUSDT)
			ph.InitialBought.Add(ph.InitialBought, hol.InitialBought)
			ph.P2PBought.Add(ph.P2PBought, hol.P2PBought)
			ph.InitSaleUSDT.Add(ph.InitSaleUSDT, hol.InitSaleUSDT)
			ph.P2PUSDT.Add(ph.P2PUSDT, hol.P2PUSDT)
			ph.OffPlatformP2P = ph.OffPlatformP2P || hol.OffPlatformP2P
			ph.Entered = earlierTime(ph.Entered, hol.Entered)
			ph.Exited = laterTime(ph.Exited, hol.Exited)
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
		pct := PercentFloat(h.TotalBalance, pr.TotalSupplyRaw)
		tier := ""
		if h.TotalBalance.Sign() > 0 {
			tier = holderTier(pct)
			pcts = append(pcts, pct)
		}
		holders = append(holders, projectHolderPayload{
			Address:          h.Address,
			PropertyNames:    slices.Sorted(maps.Keys(h.PropertyBalances)),
			Balance:          FormatBigInt(h.TotalBalance, pr.Decimal),
			WeekDeltaTokens:  formatDelta(h.WeekDeltaTokens, pr.Decimal),
			MonthDeltaTokens: formatDelta(h.MonthDeltaTokens, pr.Decimal),
			WeekDeltaUSDT:    formatDelta(h.WeekDeltaUSDT, polygonscan.USDTDecimal),
			MonthDeltaUSDT:   formatDelta(h.MonthDeltaUSDT, polygonscan.USDTDecimal),
			InitialBought:    FormatBigInt(h.InitialBought, pr.Decimal),
			P2PBought:        FormatBigInt(h.P2PBought, pr.Decimal),
			InitSaleUSDT:     FormatBigInt(h.InitSaleUSDT, polygonscan.USDTDecimal),
			P2PUSDT:          FormatBigInt(h.P2PUSDT, polygonscan.USDTDecimal),
			OffPlatformP2P:   h.OffPlatformP2P,
			SupplyPct:        pct,
			Tier:             tier,
			Entered:          formatDate(h.Entered),
			Exited:           formatDate(h.Exited),
		})
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

func earlierTime(a, b time.Time) time.Time {
	if a.IsZero() || (!b.IsZero() && b.Before(a)) {
		return b
	}
	return a
}

func laterTime(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

func formatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(timeDateOnly)
}

func holderTier(percent float64) string {
	for _, t := range holderTierThresholds {
		if percent <= t.max {
			return t.name
		}
	}
	return holderTierThresholds[len(holderTierThresholds)-1].name
}
