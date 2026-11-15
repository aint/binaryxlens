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
	InitialBought  *big.Int
	P2PBought      *big.Int
	InitSaleUSDT   *big.Int
	P2PUSDT        *big.Int
	OffPlatformP2P bool
	Entered        time.Time
	Exited         time.Time
	DailyActivity  DailyActivity
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

type DailyActivity map[Day]*Activity

type Activity struct {
	TokensIn  *big.Int
	TokensOut *big.Int
	UsdtIn    *big.Int
	UsdtOut   *big.Int
}

type Day time.Time

// NewDay creates a new time instant at 00:00 UTC
func NewDay(t time.Time) Day {
	return Day(t.UTC().Truncate(24 * time.Hour))
}

func (h *Holder) TokenBalance() *big.Int {
	tokens := big.NewInt(0)
	for _, a := range h.DailyActivity {
		tokens.Add(tokens, a.TokensIn)
		tokens.Add(tokens, a.TokensOut)
	}
	return tokens
}

func (h *Holder) TokenWeekDelta() *big.Int {
	return h.tokenDelta(7)
}

func (h *Holder) TokenMonthDelta() *big.Int {
	return h.tokenDelta(30)
}

func (h *Holder) tokenDelta(days int) *big.Int {
	now := time.Now().UTC()
	delta := big.NewInt(0)
	for i := range days {
		day := NewDay(now.AddDate(0, 0, -i))
		activity := h.DailyActivity[day]
		if activity == nil {
			continue
		}
		delta.Add(delta, activity.TokensIn)
		delta.Add(delta, activity.TokensOut)
	}
	return delta
}

func (h *Holder) UsdtWeekDelta() *big.Int {
	return h.usdtDelta(7)
}

func (h *Holder) UsdtMonthDelta() *big.Int {
	return h.usdtDelta(30)
}

func (h *Holder) usdtDelta(days int) *big.Int {
	now := time.Now().UTC()
	delta := big.NewInt(0)
	for i := range days {
		day := NewDay(now.AddDate(0, 0, -i))
		activity := h.DailyActivity[day]
		if activity == nil {
			continue
		}
		delta.Add(delta, activity.UsdtIn)
		delta.Add(delta, activity.UsdtOut)
	}
	return delta
}

func (p *Property) buildHolders() {
	holderMap := make(map[string]*Holder)
	for _, t := range p.transfers {
		v := t.Value
		day := NewDay(t.Time)

		// tokens sent from t.From (balance goes down)
		if t.From != zeroAddr0x && t.From != p.Address {
			var paid *big.Int
			if p.isP2PTransfer(t.From, t.To) {
				paid = negBigInt(t.USDT)
			}
			updateActivity(holderMap, t.From, new(big.Int).Neg(v), paid, day)
			from := holderMap[t.From]
			if from.TokenBalance().Sign() == 0 {
				from.Exited = t.Time
			}
		}
		// tokens sent to t.To (balance goes up)
		if t.To != zeroAddr0x && t.To != p.Address {
			to := holderMap[t.To]
			opening := to == nil || to.TokenBalance().Sign() == 0
			updateActivity(holderMap, t.To, v, t.USDT, day)
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
		if h.TokenBalance().Sign() == 0 && h.TokenMonthDelta().Sign() == 0 {
			continue
		}
		holders = append(holders, *h)
	}

	p.Holders = holders
}

func updateActivity(m map[string]*Holder, addr string, tokens, usdt *big.Int, day Day) {
	h := m[addr]
	if h == nil {
		h = &Holder{
			Address:        addr,
			InitialBought:  big.NewInt(0),
			P2PBought:      big.NewInt(0),
			InitSaleUSDT:   big.NewInt(0),
			P2PUSDT:        big.NewInt(0),
			DailyActivity:  make(DailyActivity),
		}
		m[addr] = h
	}

	activity := h.DailyActivity[day]
	if activity == nil {
		activity = &Activity{
			TokensIn:  big.NewInt(0),
			TokensOut: big.NewInt(0),
			UsdtOut:   big.NewInt(0),
			UsdtIn:    big.NewInt(0),
		}
		h.DailyActivity[day] = activity
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
					WeekDeltaUSDT:    new(big.Int).Set(hol.UsdtWeekDelta()),
					MonthDeltaUSDT:   new(big.Int).Set(hol.UsdtMonthDelta()),
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

			ph.PropertyBalances[property.Name] = new(big.Int).Set(hol.TokenBalance())
			ph.TotalBalance.Add(ph.TotalBalance, hol.TokenBalance())
			ph.WeekDeltaTokens.Add(ph.WeekDeltaTokens, hol.TokenWeekDelta())
			ph.MonthDeltaTokens.Add(ph.MonthDeltaTokens, hol.TokenMonthDelta())
			ph.WeekDeltaUSDT.Add(ph.WeekDeltaUSDT, hol.UsdtWeekDelta())
			ph.MonthDeltaUSDT.Add(ph.MonthDeltaUSDT, hol.UsdtMonthDelta())
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
