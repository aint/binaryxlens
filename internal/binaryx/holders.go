package binaryx

import (
	"maps"
	"math/big"
	"slices"
	"time"
)

const zeroAddr0x = "0x0000000000000000000000000000000000000000"

type Holder struct {
	Address        string
	WeekDeltaUSDT  *big.Int
	MonthDeltaUSDT *big.Int
	InitialBought  *big.Int
	P2PBought      *big.Int
	InitSaleUSDT   *big.Int
	P2PUSDT        *big.Int
	OffPlatformP2P bool
	Entered        time.Time
	Exited         time.Time
	WeeklyActivity WeeklyActivity
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
	WeeklyActivity   WeeklyActivity
}

type WeeklyActivity map[Week]*Activity

type Activity struct {
	TokensIn  *big.Int
	TokensOut *big.Int
	UsdtIn    *big.Int
	UsdtOut   *big.Int
}

type Week time.Time

// NewWeek creates a new time instant at Monday 00:00 UTC
func NewWeek(t time.Time) Week {
	t = t.UTC().Truncate(24 * time.Hour)
	offset := (int(t.Weekday()) + 6) % 7
	return Week(t.AddDate(0, 0, -offset))
}

func (a WeeklyActivity) Merge(a2 WeeklyActivity) {
	for k, v := range a2 {
		if _, exists := a[k]; !exists {
			a[k] = v
			continue
		}
		a[k].TokensIn.Add(a[k].TokensIn, v.TokensIn)
		a[k].TokensOut.Add(a[k].TokensOut, v.TokensOut)
		a[k].UsdtIn.Add(a[k].UsdtIn, v.UsdtIn)
		a[k].UsdtOut.Add(a[k].UsdtOut, v.UsdtOut)
	}
}

func (h *Holder) TokenBalance() *big.Int {
	tokens := big.NewInt(0)
	for _, v := range h.WeeklyActivity {
		tokens.Add(tokens, v.TokensIn)
		tokens.Add(tokens, v.TokensOut)
	}
	return tokens
}

func (h *Holder) TokenWeekDelta() *big.Int {
	delta := big.NewInt(0)
	week := NewWeek(time.Now().UTC())
	activity := h.WeeklyActivity[week]
	if activity == nil {
		return delta
	}
	return delta.Add(activity.TokensIn, activity.TokensOut)
}

func (h *Holder) TokenMonthDelta() *big.Int {
	now := time.Now().UTC()
	delta := big.NewInt(0)
	for i := range 5 {
		week := NewWeek(now.AddDate(0, 0, -i*7))
		activity := h.WeeklyActivity[week]
		if activity == nil {
			continue
		}
		delta.Add(activity.TokensIn, activity.TokensOut)
	}
	return delta
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
		week := NewWeek(t.Time)

		// tokens sent from t.From (balance goes down)
		if t.From != zeroAddr0x && t.From != p.Address {
			var paid *big.Int
			if p.isP2PTransfer(t.From, t.To) {
				paid = negBigInt(t.USDT)
			}
			updateHolder(holderMap, t.From, new(big.Int).Neg(v), paid, week, inWeek, inMonth)
			from := holderMap[t.From]
			if from.TokenBalance().Sign() == 0 {
				from.Exited = t.Time
			}
		}
		// tokens sent to t.To (balance goes up)
		if t.To != zeroAddr0x && t.To != p.Address {
			to := holderMap[t.To]
			opening := to == nil || to.TokenBalance().Sign() == 0
			updateHolder(holderMap, t.To, v, t.USDT, week, inWeek, inMonth)
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
		return holderMap[b].TokenBalance().Cmp(holderMap[a].TokenBalance()) // descending by balance
	})

	holders := make([]Holder, 0, len(holderMap))
	for _, address := range keys {
		h := holderMap[address]
		if h.TokenBalance().Sign() == 0 && h.TokenMonthDelta().Sign() == 0 && h.MonthDeltaUSDT.Sign() == 0 {
			continue
		}
		holders = append(holders, *h)
	}

	p.Holders = holders
}

func updateHolder(m map[string]*Holder, addr string, tokens, usdt *big.Int, week Week, inWeek, inMonth bool) {
	h := m[addr]
	if h == nil {
		h = &Holder{
			Address:        addr,
			WeekDeltaUSDT:  big.NewInt(0),
			MonthDeltaUSDT: big.NewInt(0),
			InitialBought:  big.NewInt(0),
			P2PBought:      big.NewInt(0),
			InitSaleUSDT:   big.NewInt(0),
			P2PUSDT:        big.NewInt(0),
			WeeklyActivity: make(WeeklyActivity),
		}
		m[addr] = h
	}
	if inWeek {
		if usdt != nil {
			h.WeekDeltaUSDT.Add(h.WeekDeltaUSDT, usdt)
		}
	}
	if inMonth {
		if usdt != nil {
			h.MonthDeltaUSDT.Add(h.MonthDeltaUSDT, usdt)
		}
	}

	activity := h.WeeklyActivity[week]
	if activity == nil {
		activity = &Activity{
			TokensIn:  big.NewInt(0),
			TokensOut: big.NewInt(0),
			UsdtOut:   big.NewInt(0),
			UsdtIn:    big.NewInt(0),
		}
		h.WeeklyActivity[week] = activity
	}
	if tokens.Sign() > 0 {
		activity.TokensIn.Add(activity.TokensIn, tokens)
	} else if tokens.Sign() < 0 {
		activity.TokensOut.Add(activity.TokensOut, tokens)
	}
	if usdt != nil {
		if usdt.Sign() > 0 {
			activity.UsdtIn.Add(activity.UsdtIn, usdt)
		} else if usdt.Sign() < 0 {
			activity.UsdtOut.Add(activity.UsdtOut, usdt)
		}
	}
}

func (pr *Project) BuildHolders() {
	projectHolderMap := make(map[string]*ProjectHolder)

	for _, property := range pr.Properties {
		for _, hol := range property.Holders {
			ph := projectHolderMap[hol.Address]
			if ph == nil {
				ph = &ProjectHolder{
					Address:          hol.Address,
					PropertyBalances: map[string]*big.Int{property.Name: new(big.Int).Set(hol.TokenBalance())},
					TotalBalance:     new(big.Int).Set(hol.TokenBalance()),
					WeekDeltaTokens:  new(big.Int).Set(hol.TokenWeekDelta()),
					MonthDeltaTokens: new(big.Int).Set(hol.TokenMonthDelta()),
					WeekDeltaUSDT:    new(big.Int).Set(hol.WeekDeltaUSDT),
					MonthDeltaUSDT:   new(big.Int).Set(hol.MonthDeltaUSDT),
					InitialBought:    new(big.Int).Set(hol.InitialBought),
					P2PBought:        new(big.Int).Set(hol.P2PBought),
					InitSaleUSDT:     new(big.Int).Set(hol.InitSaleUSDT),
					P2PUSDT:          new(big.Int).Set(hol.P2PUSDT),
					OffPlatformP2P:   hol.OffPlatformP2P,
					Entered:          hol.Entered,
					Exited:           hol.Exited,
					WeeklyActivity:   hol.WeeklyActivity,
				}
				projectHolderMap[hol.Address] = ph
				continue
			}

			ph.PropertyBalances[property.Name] = new(big.Int).Set(hol.TokenBalance())
			ph.TotalBalance.Add(ph.TotalBalance, hol.TokenBalance())
			ph.WeekDeltaTokens.Add(ph.WeekDeltaTokens, hol.TokenWeekDelta())
			ph.MonthDeltaTokens.Add(ph.MonthDeltaTokens, hol.TokenMonthDelta())
			ph.WeekDeltaUSDT.Add(ph.WeekDeltaUSDT, hol.WeekDeltaUSDT)
			ph.MonthDeltaUSDT.Add(ph.MonthDeltaUSDT, hol.MonthDeltaUSDT)
			ph.InitialBought.Add(ph.InitialBought, hol.InitialBought)
			ph.P2PBought.Add(ph.P2PBought, hol.P2PBought)
			ph.InitSaleUSDT.Add(ph.InitSaleUSDT, hol.InitSaleUSDT)
			ph.P2PUSDT.Add(ph.P2PUSDT, hol.P2PUSDT)
			ph.OffPlatformP2P = ph.OffPlatformP2P || hol.OffPlatformP2P
			ph.Entered = earlierTime(ph.Entered, hol.Entered)
			ph.Exited = laterTime(ph.Exited, hol.Exited)
			ph.WeeklyActivity.Merge(hol.WeeklyActivity)
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
