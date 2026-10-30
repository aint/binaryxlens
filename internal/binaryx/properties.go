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
	RentalStartActual     *YearQuarter
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
		RentalStartActual:   params.RentalStartActual,
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

var AllProperties = map[string][]*Property{
	"AWWA Hotel by Ribas": {
		newProperty(
			propertyParams{
				Name:                "AWWA Hotel by Ribas B14",
				Address:             "0x216301b87404a5839bf7b8b94c646c4eb96fec79",
				State:               Rental,
				Location:            Bali, // Canggu
				AreaM2:              33,
				Raised:              157_500,
				MarketLow:           2_800,
				MarketHigh:          3_800,
				RentalStartExpected: YearQuarter{Year: 2025, Quarter: 2},
				RentalStartActual:   &YearQuarter{Year: 2025, Quarter: 4},
			},
		),
		newProperty(
			propertyParams{
				Name:                "AWWA Hotel by Ribas B22",
				Address:             "0xe725a80f426a7d7f5734ba69ccec507251109d09",
				State:               Rental,
				Location:            Bali, // Canggu
				AreaM2:              33,
				Raised:              157_500,
				MarketLow:           2_800,
				MarketHigh:          3_800,
				RentalStartExpected: YearQuarter{Year: 2025, Quarter: 2},
				RentalStartActual:   &YearQuarter{Year: 2025, Quarter: 4},
			},
		),
		newProperty(
			propertyParams{
				Name:                "AWWA Hotel by Ribas A16",
				Address:             "0xdb8fc93a993e2ab0d9f7d520fd4e616cfb1d85fd",
				State:               Rental,
				Location:            Bali, // Canggu
				AreaM2:              33,
				Raised:              157_500,
				MarketLow:           2_800,
				MarketHigh:          3_800,
				RentalStartExpected: YearQuarter{Year: 2025, Quarter: 2},
				RentalStartActual:   &YearQuarter{Year: 2025, Quarter: 4},
			},
		),
	},
	"Aurora Villas (Bali Balance Ocean Villas)": {
		newProperty(
			propertyParams{
				Name:                "Villa Aurora 1 (Ocean Villa 3)",
				Address:             "0x1e3cf2eeaa6d5973e2da6fe03600ba55870dd69b",
				State:               Rental,
				Location:            Bali, // Uluwatu
				AreaM2:              165,
				Raised:              380_000,
				Projected:           428_500,
				MarketLow:           2_100,
				MarketHigh:          3_200,
				RentalStartExpected: YearQuarter{Year: 2025, Quarter: 4},
			},
		),
		newProperty(
			propertyParams{
				Name:                "Villa Aurora 2 (Ocean Villa 4)",
				Address:             "0x17236ed296fbd00d3dfa016879833776dd207fd6",
				State:               Rental,
				Location:            Bali, // Uluwatu
				AreaM2:              165,
				Raised:              380_000,
				Projected:           428_500,
				MarketLow:           2_100,
				MarketHigh:          3_200,
				RentalStartExpected: YearQuarter{Year: 2025, Quarter: 4},
			},
		),
	},
	"Bingin Magic Story Villas": {
		newProperty(
			propertyParams{
				Name:                "Bingin Magic Story Villa 3",
				Address:             "0xe5f846592a58bcfce912bc6fc594649397b6f519",
				State:               Rental,
				Location:            Bali, // Canggu
				AreaM2:              115,
				Raised:              249_000,
				Projected:           280_000,
				MarketLow:           2_100,
				MarketHigh:          3_200,
				RentalStartExpected: YearQuarter{Year: 2025, Quarter: 4},
				RentalStartActual:   &YearQuarter{Year: 2026, Quarter: 3},
			},
		),
	},
	"Bubbles Boutique Complex": {
		newProperty(
			propertyParams{
				Name:                "Bubbles Boutique Complex",
				Address:             "0xC1EA0Ccd94F17Ec0580DD57A34C2B521360ad4b1",
				State:               Rental,
				Location:            Bali, //Canggu
				AreaM2:              555,
				Raised:              1_250_000,
				MarketLow:           2_100,
				MarketHigh:          2_700,
				RentalStartExpected: YearQuarter{Year: 2025, Quarter: 4},
				RentalStartActual:   &YearQuarter{Year: 2025, Quarter: 4},
				RentalProfitMonths: []RentalProfitMonth{
					{Month: time.Date(2024, time.August, 1, 0, 0, 0, 0, time.UTC), USDT: big.NewInt(1828000000)},
					{Month: time.Date(2024, time.September, 1, 0, 0, 0, 0, time.UTC), USDT: big.NewInt(2968000000)},
					{Month: time.Date(2024, time.October, 1, 0, 0, 0, 0, time.UTC), USDT: big.NewInt(2290000000)},
					{Month: time.Date(2024, time.November, 1, 0, 0, 0, 0, time.UTC), USDT: big.NewInt(2882000000)},
					{Month: time.Date(2024, time.December, 1, 0, 0, 0, 0, time.UTC), USDT: big.NewInt(6870000000)},
					{Month: time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC), USDT: big.NewInt(6870000000)},
					{Month: time.Date(2025, time.February, 1, 0, 0, 0, 0, time.UTC), USDT: big.NewInt(8319000000)},
					{Month: time.Date(2025, time.March, 1, 0, 0, 0, 0, time.UTC), USDT: big.NewInt(11298000000)},
				},
			},
		),
	},
	"CASCADE Villas": {
		newProperty(
			propertyParams{
				Name:                "CASCADE Villa 2",
				Address:             "0x5e55b3e941f42732f1b941f2f673dc8811355e5e",
				State:               Rental,
				Location:            Bali, // Canggu
				AreaM2:              176,
				Raised:              265_000,
				Projected:           300_000,
				MarketLow:           1_700,
				MarketHigh:          2_400,
				RentalStartExpected: YearQuarter{Year: 2026, Quarter: 1},
			},
		),
		newProperty(
			propertyParams{
				Name:                "CASCADE Villa 3",
				Address:             "0xd5551375d5ba01ddbcb38d20ac40671f26e6ada5",
				State:               Rental,
				Location:            Bali, // Canggu
				AreaM2:              176,
				Raised:              265_000,
				Projected:           300_000,
				MarketLow:           1_700,
				MarketHigh:          2_400,
				RentalStartExpected: YearQuarter{Year: 2026, Quarter: 1},
			},
		),
	},
	"CEMAGI Units": {
		newProperty(
			propertyParams{
				Name:                "CEMAGI Unit 3.44",
				Address:             "0x852b6995628b760c84bdd02bc143b48288d4dd3a",
				State:               Construction,
				Location:            Bali, // Canggu
				AreaM2:              26,
				Raised:              112_000,
				Projected:           130_000,
				MarketLow:           3_000,
				MarketHigh:          4_500,
				RentalStartExpected: YearQuarter{Year: 2026, Quarter: 2},
			},
		),
		newProperty(
			propertyParams{
				Name:                "CEMAGI Unit 3.46",
				Address:             "0x2b7dca2c2bafdb1dac0e01068091590fbe09e478",
				State:               Construction,
				Location:            Bali, // Canggu
				AreaM2:              26,
				Raised:              112_000,
				Projected:           130_000,
				MarketLow:           3_000,
				MarketHigh:          4_500,
				RentalStartExpected: YearQuarter{Year: 2026, Quarter: 2},
			},
		),
	},
	"Dukley Glamping 1": {
		newProperty(
			propertyParams{
				Name:                "Dukley Glamping 1",
				Address:             "0xad4f81d0f2f626a6ea29864f488604e6b5360e2a",
				State:               Rental,
				Location:            Montenegro, // Kolasin
				AreaM2:              240,
				Raised:              940_000,
				Projected:           1_050_000,
				MarketLow:           2_300,
				MarketHigh:          3_500,
				RentalStartExpected: YearQuarter{Year: 2026, Quarter: 4},
			},
		),
	},
	"Mountain Retreat by Dukley": {
		newProperty(
			propertyParams{
				Name:                "Mountain Retreat by Dukley",
				Address:             "0x51343ee93059cbb11c4bf969a643e09117b3af6b",
				State:               Redeemed,
				Location:            Montenegro, // Kolasin
				AreaM2:              73,
				Raised:              385_000,
				Projected:           440_000,
				RentalStartExpected: YearQuarter{Year: 2025, Quarter: 1},
			},
		),
	},
	"Ecoverse Suites": {
		newProperty(
			propertyParams{
				Name:                "Ecoverse Suite",
				Address:             "0x30ed65e470be4f351abf5311769505e3f977deca",
				State:               Rental,
				Location:            Bali, // Canggu
				AreaM2:              41,
				Raised:              178_500,
				Projected:           214_200,
				MarketLow:           2_500,
				MarketHigh:          3_500,
				RentalStartExpected: YearQuarter{Year: 2025, Quarter: 3},
				RentalStartActual:   &YearQuarter{Year: 2026, Quarter: 3},
			},
		),
	},
	"Hayat Green Tower": {
		newProperty(
			propertyParams{
				Name:                "Hayat Green Tower",
				Address:             "0xF9d43a9F7Fc7ee7ac47fE96De20e545243450b27",
				State:               Rental,
				Location:            Turkey, // Avsallar, Alanya
				AreaM2:              105,
				Raised:              305_000,
				MarketLow:           1_000,
				MarketHigh:          1_300,
				RentalStartExpected: YearQuarter{Year: 2023, Quarter: 3},
				RentalStartActual:   &YearQuarter{Year: 2024, Quarter: 3},
			},
		),
	},
	"Kammara Loft": {
		newProperty(
			propertyParams{
				Name:                "Kammara Loft",
				Address:             "0xB1B987FF1F317A47876185dE4dE9C430823Ad8c5",
				State:               Rental,
				Location:            Bali, // Canggu
				AreaM2:              70,
				Raised:              200_000,
				MarketLow:           2_200,
				MarketHigh:          3_000,
				RentalStartExpected: YearQuarter{Year: 2023, Quarter: 4},
				RentalStartActual:   &YearQuarter{Year: 2024, Quarter: 4},
			},
		),
	},
	"Kammora Living": {
		newProperty(
			propertyParams{
				Name:                "Kammora Living",
				Address:             "0x8389AcD0e05990eF0e087A3BCed3E9C5443d0455",
				State:               Rental,
				Location:            Bali, // Canggu
				AreaM2:              49,
				Raised:              150_000,
				MarketLow:           2_200,
				MarketHigh:          3_000,
				RentalStartExpected: YearQuarter{Year: 2024, Quarter: 3},
				RentalStartActual:   &YearQuarter{Year: 2024, Quarter: 3},
			},
		),
	},
	"Karra Loft": {
		newProperty(
			propertyParams{
				Name:                "Karra Loft 3A",
				Address:             "0x27Ceb34AC7545F78A97C0500465aCE9fA10570af",
				State:               Rental,
				Location:            Bali, // Canggu
				AreaM2:              68,
				Raised:              220_000,
				MarketLow:           2_200,
				MarketHigh:          3_000,
				RentalStartExpected: YearQuarter{Year: 2023, Quarter: 3},
				RentalStartActual:   &YearQuarter{Year: 2023, Quarter: 3},
			},
		),
		newProperty(
			propertyParams{
				Name:                "Karra Loft 5",
				Address:             "0x2d02E704174635F5E88E17995C3a5E29f283C033",
				State:               Rental,
				Location:            Bali, // Canggu
				AreaM2:              70,
				Raised:              220_000,
				MarketLow:           2_200,
				MarketHigh:          3_000,
				RentalStartExpected: YearQuarter{Year: 2023, Quarter: 3},
				RentalStartActual:   &YearQuarter{Year: 2023, Quarter: 3},
			},
		),
	},
	"La Casa Española Villas": {
		newProperty(
			propertyParams{
				Name:                "La Casa Española Villa 4",
				Address:             "0x7b592d8bb722324f75af834c23e6ad2058b168e1",
				State:               Construction,
				Location:            Bali, // Bingin, Uluwatu
				AreaM2:              166,          // binaryx website: 230
				Raised:              385_000,
				Projected:           460_000,
				MarketLow:           2_100,
				MarketHigh:          3_200,
				RentalStartExpected: YearQuarter{Year: 2026, Quarter: 4},
			},
		),
		newProperty(
			propertyParams{
				Name:                "La Casa Española Villa 6",
				Address:             "0xdd36b686a5ff910b5074e3f5483135f19e49f02c",
				State:               Construction,
				Location:            Bali, // Bingin, Uluwatu
				AreaM2:              161,          // binaryx: 186
				Raised:              350_000,
				Projected:           425_000,
				MarketLow:           2_100,
				MarketHigh:          3_200,
				RentalStartExpected: YearQuarter{Year: 2026, Quarter: 4},
			},
		),
		newProperty(
			propertyParams{
				Name:                "La Casa Española Villa 8",
				Address:             "0x223270bbbe4f6dac0dc3e57d985116bdc50616ee",
				State:               Construction,
				Location:            Bali, // Bingin, Uluwatu
				AreaM2:              171,          // binaryx: 211
				Raised:              350_000,
				Projected:           425_000,
				MarketLow:           2_100,
				MarketHigh:          3_200,
				RentalStartExpected: YearQuarter{Year: 2026, Quarter: 4},
			},
		),
		newProperty(
			propertyParams{
				Name:                "La Casa Española Villa 9",
				Address:             "0x89ebdfaf79308871a24c6992232984b3c84af9a8",
				State:               Construction,
				Location:            Bali, // Bingin, Uluwatu
				AreaM2:              200,          // binaryx: 241
				Raised:              385_000,
				Projected:           460_000,
				MarketLow:           2_100,
				MarketHigh:          3_200,
				RentalStartExpected: YearQuarter{Year: 2026, Quarter: 4},
			},
		),
	},
	"Oasis Royal Collection": {
		newProperty(
			propertyParams{
				Name:                "Oasis Royal Collection 11a",
				Address:             "0xa26f11748ed29b3fd62e1d8e231d277a0980fb12",
				State:               Rental,
				Location:            Bali, //Pererenan, north of Canggu
				AreaM2:              40,
				Raised:              120_000,
				Projected:           144_000,
				MarketLow:           2_500,
				MarketHigh:          3_500,
				RentalStartExpected: YearQuarter{Year: 2025, Quarter: 4},
			},
		),
		newProperty(
			propertyParams{
				Name:                "Oasis Royal Collection 18b",
				Address:             "0x1dac5a4a0e566fb2674a6b7e1cdaf2c07716eeed",
				State:               Rental,
				Location:            Bali, //Pererenan, north of Canggu
				AreaM2:              40,
				Raised:              120_000,
				Projected:           144_000,
				MarketLow:           2_500,
				MarketHigh:          3_500,
				RentalStartExpected: YearQuarter{Year: 2025, Quarter: 4},
			},
		),
	},
	"Onyx PARQ Resort 61": {
		newProperty(
			propertyParams{
				Name:                "Onyx PARQ Resort 61",
				Address:             "0xA07DB641FC95067a2Fe68b6224a9dD39564bFd57",
				State:               Rental,
				Location:            Bali, // Parq, Ubud
				AreaM2:              64,
				Raised:              171_612,
				MarketLow:           1_800,
				MarketHigh:          2_600,
				RentalStartExpected: YearQuarter{Year: 2023, Quarter: 2},
				RentalStartActual:   &YearQuarter{Year: 2023, Quarter: 2},
			},
		),
	},
	"Roots Villas": {
		newProperty(
			propertyParams{
				Name:                "Roots Villa 1",
				Address:             "0xbde380b4cc582d440255ebd89ff1839dcfad5d7b",
				State:               Construction,
				Location:            Bali, // Ubud
				AreaM2:              161,
				Raised:              303_000,
				Projected:           353_000,
				MarketLow:           1_800,
				MarketHigh:          2_600,
				RentalStartExpected: YearQuarter{Year: 2026, Quarter: 3},
			},
		),
		newProperty(
			propertyParams{
				Name:                "Roots Villa 3",
				Address:             "0xc0a4b2e29bd44d3b798a02edc039711f03572739",
				State:               Construction,
				Location:            Bali, // Ubud
				AreaM2:              188,
				Raised:              350_000,
				Projected:           405_000,
				MarketLow:           1_800,
				MarketHigh:          2_600,
				RentalStartExpected: YearQuarter{Year: 2026, Quarter: 3},
			},
		),
		newProperty(
			propertyParams{
				Name:                "Roots Villa 4",
				Address:             "0xb2b9f922c0494dbf08636b1dbcf6fcba0878a605",
				State:               Construction,
				Location:            Bali, // Ubud
				AreaM2:              188,
				Raised:              350_000,
				Projected:           405_000,
				MarketLow:           1_800,
				MarketHigh:          2_600,
				RentalStartExpected: YearQuarter{Year: 2026, Quarter: 3},
			},
		),
		newProperty(
			propertyParams{
				Name:                "Roots Villa 5",
				Address:             "0x0ef68e86c3c9bc6187c69770053919e6b35991f6",
				State:               Construction,
				Location:            Bali, // Ubud
				AreaM2:              188,
				Raised:              350_000,
				Projected:           405_000,
				MarketLow:           1_800,
				MarketHigh:          2_600,
				RentalStartExpected: YearQuarter{Year: 2026, Quarter: 3},
			},
		),
	},
	"Taryan Dragon Jungle View": {
		newProperty(
			propertyParams{
				Name:                "Taryan Dragon Jungle View",
				Address:             "0x4bd4d7003a6ce76b9ad3ee364a29801c170b1ff5",
				State:               Construction,
				Location:            Bali, // Canggu
				AreaM2:              55,
				Projected:           340_000,
				MarketLow:           3_500,
				MarketHigh:          5_000,
				RentalStartExpected: YearQuarter{Year: 2027, Quarter: 4},
			},
		),
	},
	"Tropical Loft Villas": {
		newProperty(
			propertyParams{
				Name:                "Tropical Loft Villa 2",
				Address:             "0x4b17845F255cC19dB2612ab8577Ea1e0852BBBd7",
				State:               Rental,
				Location:            Bali, //Padonan, Canggu
				AreaM2:              126,
				Raised:              225_750,
				MarketLow:           1_850,
				MarketHigh:          2_300,
				RentalStartExpected: YearQuarter{Year: 2025, Quarter: 1},
				RentalStartActual:   &YearQuarter{Year: 2025, Quarter: 1},
			},
		),
		newProperty(
			propertyParams{
				Name:                "Tropical Loft Villa 3",
				Address:             "0x56467B7E0cF2116A1C7664eE60db77ED24709293",
				State:               Rental,
				Location:            Bali, //Padonan, Canggu
				AreaM2:              126,
				Raised:              225_750,
				MarketLow:           1_850,
				MarketHigh:          2_300,
				RentalStartExpected: YearQuarter{Year: 2025, Quarter: 2},
				RentalStartActual:   &YearQuarter{Year: 2025, Quarter: 2},
			},
		),
		newProperty(
			propertyParams{
				Name:                "Tropical Loft Villa 4",
				Address:             "0x6a6F5681678Cc599d4d4Ae55270070406561DCa7",
				State:               Rental,
				Location:            Bali, //Padonan, Canggu
				AreaM2:              126,
				Raised:              225_750,
				MarketLow:           1_850,
				MarketHigh:          2_300,
				RentalStartExpected: YearQuarter{Year: 2025, Quarter: 2},
				RentalStartActual:   &YearQuarter{Year: 2025, Quarter: 2},
			},
		),
	},
	"Vesna Townhouse": {
		newProperty(
			propertyParams{
				Name:                "Vesna Townhouse",
				Address:             "0x09558935e9cA1c4985F96163b62Fd850616F6e33",
				State:               Rental,
				Location:            Bali, // Berawa area of Canggu
				AreaM2:              64,
				Raised:              220_000,
				MarketLow:           1_900,
				MarketHigh:          2_500,
				RentalStartExpected: YearQuarter{Year: 2024, Quarter: 2},
				RentalStartActual:   &YearQuarter{Year: 2024, Quarter: 2},
			},
		),
	},
}