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
	Tokens         *Balance
	USDT           *Balance
	OffPlatformP2P bool
	Entered        time.Time
	Exited         time.Time
	DailyActivity  DailyActivity
}

type ProjectHolder struct {
	Address          string
	PropertyBalances map[string]*big.Int
	Tokens           *Balance
	USDT             *Balance
	WeekDeltaTokens  *big.Int
	MonthDeltaTokens *big.Int
	WeekDeltaUSDT    *big.Int
	MonthDeltaUSDT   *big.Int
	OffPlatformP2P   bool
	Entered          time.Time
	Exited           time.Time
}

type Balance struct {
	Total    *big.Int
	InitSale *big.Int
	P2P      *big.Int
	Max      *big.Int
}

func newBalance() *Balance {
	return &Balance{
		Total: big.NewInt(0), InitSale: big.NewInt(0), P2P: big.NewInt(0), Max: big.NewInt(0),
	}
}

type DailyActivity map[Day]*Activity

type Activity struct {
	TokensIn  *big.Int
	TokensOut *big.Int
	UsdtIn    *big.Int
	UsdtOut   *big.Int
}

func newActivity() *Activity {
	return &Activity{
		TokensIn: big.NewInt(0), TokensOut: big.NewInt(0), UsdtIn: big.NewInt(0), UsdtOut: big.NewInt(0),
	}
}

type Day time.Time

// NewDay creates a new time instant at 00:00 UTC
func NewDay(t time.Time) Day {
	return Day(t.UTC().Truncate(24 * time.Hour))
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
			updateBalances(holderMap, p, t.From, t.To, new(big.Int).Neg(v), paid, day)
			from := holderMap[t.From]
			if from.Tokens.Total.Sign() == 0 {
				from.Exited = t.Time
			}
		}
		// tokens sent to t.To (balance goes up)
		if t.To != zeroAddr0x && t.To != p.Address {
			to := holderMap[t.To]
			opening := to == nil || to.Tokens.Total.Sign() == 0
			updateBalances(holderMap, p, t.From, t.To, v, t.USDT, day)
			to = holderMap[t.To]
			if opening {
				to.Entered = t.Time
				to.Exited = time.Time{}
			}
		}
	}

	keys := slices.Collect(maps.Keys(holderMap))
	slices.SortFunc(keys, func(a, b string) int {
		return holderMap[b].Tokens.Total.Cmp(holderMap[a].Tokens.Total) // descending by balance
	})

	holders := make([]Holder, 0, len(holderMap))
	for _, address := range keys {
		h := holderMap[address]
		if h.Tokens.Total.Sign() == 0 && h.TokenMonthDelta().Sign() == 0 {
			continue
		}
		holders = append(holders, *h)
	}

	p.Holders = holders
}

func updateBalances(m map[string]*Holder, p *Property, from, to string, tokens, usdt *big.Int, day Day) {
	addr := to
	if tokens.Sign() < 0 {
		addr = from
	}
	h := m[addr]
	if h == nil {
		h = &Holder{
			Address:       addr,
			Tokens:        newBalance(),
			USDT:          newBalance(),
			DailyActivity: make(DailyActivity),
		}
		m[addr] = h
	}

	activity := h.DailyActivity[day]
	if activity == nil {
		activity = newActivity()
		h.DailyActivity[day] = activity
	}
	if tokens.Sign() > 0 {
		activity.TokensIn.Add(activity.TokensIn, tokens)
		if p.isInitialSale(from) {
			h.Tokens.InitSale.Add(h.Tokens.InitSale, tokens)
			if usdt != nil {
				h.USDT.InitSale.Add(h.USDT.InitSale, usdt)
			}
		} else if p.isP2PTransfer(from, to) {
			h.Tokens.P2P.Add(h.Tokens.P2P, tokens)
			if usdt != nil {
				h.USDT.P2P.Add(h.USDT.P2P, usdt)
			}
			// Wallet-to-wallet with no platform payment.
			if usdt == nil || usdt.Sign() == 0 {
				h.OffPlatformP2P = true
			}
		}
	} else if tokens.Sign() < 0 {
		activity.TokensOut.Add(activity.TokensOut, tokens)
	}
	if tokens.Sign() != 0 {
		h.Tokens.Total.Add(h.Tokens.Total, tokens)
		if h.Tokens.Total.Cmp(h.Tokens.Max) > 0 {
			h.Tokens.Max.Set(h.Tokens.Total)
		}
	}
	if usdt != nil {
		if usdt.Sign() > 0 {
			activity.UsdtIn.Add(activity.UsdtIn, usdt)
		} else if usdt.Sign() < 0 {
			activity.UsdtOut.Add(activity.UsdtOut, usdt)
		}
		if usdt.Sign() != 0 {
			h.USDT.Total.Add(h.USDT.Total, usdt)
			if h.USDT.Total.Cmp(h.USDT.Max) > 0 {
				h.USDT.Max.Set(h.USDT.Total)
			}
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
					PropertyBalances: map[string]*big.Int{},
					Tokens:           newBalance(),
					USDT:             newBalance(),
					WeekDeltaTokens:  big.NewInt(0),
					MonthDeltaTokens: big.NewInt(0),
					WeekDeltaUSDT:    big.NewInt(0),
					MonthDeltaUSDT:   big.NewInt(0),
				}
				projectHolderMap[hol.Address] = ph
			}

			ph.PropertyBalances[property.Name] = new(big.Int).Set(hol.Tokens.Total)
			addBalance(ph.Tokens, hol.Tokens)
			addBalance(ph.USDT, hol.USDT)
			ph.WeekDeltaTokens.Add(ph.WeekDeltaTokens, hol.TokenWeekDelta())
			ph.MonthDeltaTokens.Add(ph.MonthDeltaTokens, hol.TokenMonthDelta())
			ph.WeekDeltaUSDT.Add(ph.WeekDeltaUSDT, hol.UsdtWeekDelta())
			ph.MonthDeltaUSDT.Add(ph.MonthDeltaUSDT, hol.UsdtMonthDelta())
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
		return b.Tokens.Total.Cmp(a.Tokens.Total)
	})

	pr.Holders = projectHolders
}

func addBalance(dst, src *Balance) {
	dst.Total.Add(dst.Total, src.Total)
	dst.InitSale.Add(dst.InitSale, src.InitSale)
	dst.P2P.Add(dst.P2P, src.P2P)
	if src.Max.Cmp(dst.Max) > 0 {
		dst.Max.Set(src.Max)
	}
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
