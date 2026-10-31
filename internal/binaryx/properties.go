package binaryx

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/aint/binaryxlens/internal/polygonscan"
)

type Property struct {
	Name                    string
	Address                 string
	State                   RealEstateState
	RentalStartExpected     YearQuarter
	RentalStartActual       *YearQuarter
	AreaM2                  uint
	CurrentValue            Value
	RaisedValue             Value
	ProjectedValue          Value
	MarketLowValue          uint
	MarketHighValue         uint
	Location                Location
	transfers               []transfer
	issuanceModel           IssuanceModel
	InitialSaleWeeklyPoints []WeeklyPoint
	P2PSaleWeeklyPoints     []WeeklyPoint
	RentalProfitMonths      []RentalProfitMonth
	ETAs                    []ETA
	Holders                 []Holder
	TotalSupplyRaw          *big.Int
	BoughtRaw               *big.Int
	RemainingRaw            *big.Int
	Decimal                 uint8
}

type propertyParams struct {
	Name                  string
	Address               string
	State                 RealEstateState
	Location              Location
	AreaM2                uint
	Raised, Projected     uint
	MarketLow, MarketHigh uint
	RentalStartExpected   YearQuarter
	RentalProfitMonths    []RentalProfitMonth
}

func newProperty(params propertyParams) *Property {

	p := &Property{
		Name:                params.Name,
		Address:             strings.ToLower(params.Address),
		State:               params.State,
		Location:            params.Location,
		AreaM2:              params.AreaM2,
		MarketLowValue:      params.MarketLow,
		MarketHighValue:     params.MarketHigh,
		RentalStartExpected: params.RentalStartExpected,
		RentalProfitMonths:  params.RentalProfitMonths,
	}

	p.RaisedValue = NewValue(params.Raised, p.calculateValueDelta)
	p.ProjectedValue = NewValue(params.Projected, p.calculateValueDelta)

	return p
}

// Init fetches the property's transfers and builds its stats. p2pTrades holds
// P2P trades of all properties, keyed by token address.
func (p *Property) Init(client *polygonscan.Client, scanPause time.Duration, p2pTrades []P2PTrade, monthlyRentalProfit []RentalProfitMonth) error {
	p.RentalProfitMonths = append(p.RentalProfitMonths, monthlyRentalProfit...)

	if len(p.RentalProfitMonths) > 0 {
		p.RentalStartActual = new(NewYearQuarter(p.RentalProfitMonths[0].Month))
	}

	tokenTransfers, err := client.FetchAllTokenTransfers(p.Address, 1000, scanPause)
	if err != nil {
		return fmt.Errorf("fetch token transfers: %v", err)
	}
	if len(tokenTransfers) == 0 {
		// TODO: mark as new one instead of returning error
		return fmt.Errorf("no transactions found")
	}
	p.transfers, err = newTransfers(tokenTransfers)
	if err != nil {
		return fmt.Errorf("parse token transfers: %v", err)
	}
	p.resolveIssuanceModel()

	usdtTokenTransfers, err := client.FetchUSDTTransfers(p.Address, 1000, scanPause)
	if err != nil {
		return fmt.Errorf("fetch usdt transfers: %v", err)
	}
	usdtTransfers, err := newTransfers(usdtTokenTransfers)
	if err != nil {
		return fmt.Errorf("parse usdt transfers: %v", err)
	}
	p.attachUSDT(usdtTransfers, p2pTrades)

	err = p.extractDecimal(tokenTransfers[0])
	if err != nil {
		return fmt.Errorf("extract decimal: %v", err)
	}

	p.TotalSupplyRaw, err = client.GetTotalSupply(p.Address)
	if err != nil {
		return fmt.Errorf("get total supply: %v", err)
	}
	if p.State == Redeemed {
		p.TotalSupplyRaw = calculateIssuedSupply(p.transfers)
	}
	if p.TotalSupplyRaw.Sign() == 0 {
		return errors.New("total supply is zero")
	}

	p.calculateBoughtRaw()
	p.RemainingRaw = new(big.Int).Sub(p.TotalSupplyRaw, p.BoughtRaw)

	p.calcualteCurrentValue()
	p.calculateRaisedValue()

	p.buildHolders()
	p.buildInitialSaleWeeklySeries()
	p.buildP2PSaleWeeklySeries()

	err = p.calculateMovingAverageETA()
	if err != nil {
		return fmt.Errorf("calculate ETAs: %v", err)
	}

	fmt.Printf("Property %q initialized\n", p.Name)

	return nil
}

func (p *Property) calcualteCurrentValue() {
	if p.RemainingRaw.Sign() > 0 {
		return
	}

	var usdt, tokens *big.Int
	for _, t := range p.transfers {
		if !p.isP2PTransfer(t.From, t.To) || t.USDT == nil {
			continue
		}
		usdt = t.USDT
		tokens = t.Value
	}
	if usdt == nil || tokens == nil {
		return
	}
	price := BigIntToFloat(usdt, polygonscan.USDTDecimal) / BigIntToFloat(tokens, p.Decimal) * BigIntToFloat(p.TotalSupplyRaw, p.Decimal)
	p.CurrentValue = NewValue(uint(price), p.calculateValueDelta)
}

func (p *Property) calculateRaisedValue() { // TODO: calculate raised for construction assets?
	if p.State != Construction {
		return
	}
	raised := new(big.Int)
	for _, t := range p.transfers {
		if !p.isInitialSale(t.From) || t.USDT == nil || t.USDT.Sign() <= 0 {
			continue
		}
		raised.Add(raised, t.USDT)
	}
	if raised.Sign() == 0 {
		return
	}

	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(polygonscan.USDTDecimal)), nil)
	raised.Add(raised, new(big.Int).Rsh(scale, 1))
	whole := raised.Quo(raised, scale)
	if !whole.IsUint64() {
		return
	}

	p.RaisedValue = NewValue(uint(whole.Int64()), p.calculateValueDelta)
}

func (p *Property) calculateValueDelta(price uint) (float64, bool) {
	if price == 0 || p.AreaM2 == 0 {
		return 0, false
	}
	avg := float64(p.MarketLowValue+p.MarketHighValue) / 2
	if avg == 0 {
		return 0, false
	}
	return (float64(price)/float64(p.AreaM2) - avg) / avg, true
}

func (p *Property) resolveIssuanceModel() {
	p.issuanceModel = MintOnPurchase
	for _, t := range p.transfers {
		if t.From == p.Address {
			p.issuanceModel = Escrow
			return
		}
	}
}

func (p *Property) calculateBoughtRaw() {
	boughtAmount := big.NewInt(0)
	for _, t := range p.transfers {
		if p.isInitialSale(t.From) {
			boughtAmount.Add(boughtAmount, t.Value)
		}
	}

	p.BoughtRaw = boughtAmount
}

// purchase is one buyer's purchase in one tx.
// It joins the two legs of that tx: USDT buyer→contract and tokens contract→buyer
// for an initial sale, USDT buyer→seller and tokens seller→buyer for P2P.
type purchase struct {
	txHash string
	buyer  string
}

func (p *Property) attachUSDT(initTransfers []transfer, p2pTrades []P2PTrade) {
	addPaid := func(paid map[purchase]*big.Int, hash, buyer string, amount *big.Int) {
		k := purchase{txHash: hash, buyer: buyer}
		if paid[k] == nil {
			paid[k] = new(big.Int)
		}
		paid[k].Add(paid[k], amount)
	}

	paid := make(map[purchase]*big.Int)
	for _, t := range p2pTrades {
		addPaid(paid, t.Hash, t.Buyer, t.USDT)
	}
	p.setPaidUSDT(paid, func(t transfer) bool { return p.isP2PTransfer(t.From, t.To) })

	paid = make(map[purchase]*big.Int)
	for _, t := range initTransfers {
		if t.To != p.Address {
			continue
		}
		addPaid(paid, t.Hash, t.From, t.Value)
	}
	p.setPaidUSDT(paid, func(t transfer) bool { return p.isInitialSale(t.From) })
}

// setPaidUSDT sets USDT on each transfer that isPurchase accepts and paid has a payment for.
func (p *Property) setPaidUSDT(paid map[purchase]*big.Int, isPurchase func(transfer) bool) {
	for i, t := range p.transfers {
		if !isPurchase(t) {
			continue
		}
		k := purchase{txHash: t.Hash, buyer: t.To}
		if v := paid[k]; v != nil {
			p.transfers[i].USDT = v
			// one payment may cover several sale transfers in the same tx
			delete(paid, k)
		}
	}
}

func calculateIssuedSupply(transfers []transfer) *big.Int {
	supply, maxSupply := big.NewInt(0), big.NewInt(0)
	for _, t := range transfers {
		if t.From == zeroAddr0x {
			supply.Add(supply, t.Value)
		}
		if t.To == zeroAddr0x {
			supply.Sub(supply, t.Value)
		}
		if supply.Cmp(maxSupply) > 0 {
			maxSupply.Set(supply)
		}
	}
	return maxSupply
}

// isInitialSale reports whether from is an initial sale (not wallet-to-wallet).
// Escrow: only contract→buyer transfers count; the initial 0x0 mint is inventory, not a sale.
// Mint-on-purchase: count 0x0 mints.
func (p *Property) isInitialSale(from string) bool {
	if from == p.Address {
		return true
	}
	if from == zeroAddr0x && p.issuanceModel == MintOnPurchase {
		return true
	}
	return false
}

// isP2PTransfer reports wallet-to-wallet moves after initial sale.
func (p *Property) isP2PTransfer(from, to string) bool {
	if from == zeroAddr0x || to == zeroAddr0x || from == p.Address || to == p.Address {
		return false
	}
	return true
}

func (p *Property) P2PTxCount() int {
	count := 0
	for _, t := range p.transfers {
		if p.isP2PTransfer(t.From, t.To) {
			count++
		}
	}
	return count
}

func (p *Property) extractDecimal(tx polygonscan.TokenTransfer) error {
	decimalStr := strings.TrimSpace(tx.TokenDecimal)
	if decimalStr == "" {
		return errors.New("decimal missing")
	}
	decimal, err := strconv.ParseUint(decimalStr, 10, 8)
	if err != nil {
		return fmt.Errorf("parse decimal %q: %w", decimalStr, err)
	}

	p.Decimal = uint8(decimal)

	return nil
}

func (p *Property) RentalStartDelay(now time.Time) (int, bool) {
	if p.State != Rental || p.RentalStartExpected.Quarter < 1 {
		return 0, false
	}
	actual := NewYearQuarter(now)
	if p.RentalStartActual != nil {
		actual = *p.RentalStartActual
	}
	return actual.QuartersSince(p.RentalStartExpected), true
}
