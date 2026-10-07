package internal

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/aint/binaryxlens/internal/polygonscan"
)

type PropertyType int

const (
	PropertyTypeConstruction PropertyType = iota
	PropertyTypeRental
	PropertyTypeRedeemed
)

func (t PropertyType) Label() string {
	switch t {
	case PropertyTypeConstruction:
		return "construction"
	case PropertyTypeRental:
		return "rental"
	case PropertyTypeRedeemed:
		return "redeemed"
	default:
		return "unknown"
	}
}

// IssuanceModel is how initial sales leave issuer control.
type IssuanceModel int

const (
	IssuanceMintOnPurchase IssuanceModel = iota // 0x0 → buyer on each sale
	IssuanceEscrow                              // contract → buyer; 0x0 → contract is inventory
)

type ValueTier string

const (
	ValueTierFair        ValueTier = "fair"
	ValueTierRich        ValueTier = "rich"
	ValueTierOverpriced  ValueTier = "overpriced"
	ValueTierUnderpriced ValueTier = "underpriced"
)

type Property struct {
	Name                   string
	Address                string
	Type                   PropertyType
	RentalStartExpected    YearQuarter
	RentalStartActual      *YearQuarter
	AreaM2                 uint
	BinaryxPrice           uint
	MarketLowValue         uint
	MarketHighValue        uint
	EstimatedValueTier     ValueTier
	transfers              []transfer
	issuanceModel          IssuanceModel
	InitialSaleDailyPoints []DailyPoint
	P2PSaleWeeklyPoints    []WeeklyPoint
	ETAs                   []ETA
	Holders                []Holder
	TotalSupplyRaw         *big.Int
	BoughtRaw              *big.Int
	RemainingRaw           *big.Int
	Decimal                uint8
}

type YearQuarter struct {
	Year    int
	Quarter int // 1..4
}

func (yq YearQuarter) String() string {
	return fmt.Sprintf("%d Q%d", yq.Year, yq.Quarter)
}

func (yq YearQuarter) ordinal() int {
	return yq.Year*4 + yq.Quarter - 1
}

// Init fetches the property's transfers and builds its stats. p2pTrades holds
// P2P trades of all properties, keyed by token address.
func (p *Property) Init(client *polygonscan.Client, scanPause time.Duration, p2pTrades []P2PTrade) error {
	p.Address = strings.ToLower(p.Address)
	p.estimateValueTier()

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
	if p.Type == PropertyTypeRedeemed {
		p.TotalSupplyRaw = calculateIssuedSupply(p.transfers)
	}
	if p.TotalSupplyRaw.Sign() == 0 {
		return errors.New("total supply is zero")
	}

	p.calculateBoughtRaw()
	p.RemainingRaw = new(big.Int).Sub(p.TotalSupplyRaw, p.BoughtRaw)

	p.buildHolders()
	p.buildInitialSaleDailySeries()
	p.buildP2PSaleWeeklySeries()

	err = p.calculateMovingAverageETA()
	if err != nil {
		return fmt.Errorf("calculate ETAs: %v", err)
	}

	fmt.Printf("Property %q initialized\n", p.Name)

	return nil
}

func (p *Property) estimateValueTier() {
	if p.AreaM2 == 0 {
		return
	}
	avg := float64(p.MarketLowValue+p.MarketHighValue) / 2
	if avg == 0 {
		return
	}
	delta := (float64(p.BinaryxPrice)/float64(p.AreaM2) - avg) / avg
	switch {
	case delta < -0.10:
		p.EstimatedValueTier = ValueTierUnderpriced
	case delta <= 0.10:
		p.EstimatedValueTier = ValueTierFair
	case delta <= 0.20:
		p.EstimatedValueTier = ValueTierRich
	default:
		p.EstimatedValueTier = ValueTierOverpriced
	}
}

func (p *Property) resolveIssuanceModel() {
	p.issuanceModel = IssuanceMintOnPurchase
	for _, t := range p.transfers {
		if t.From == p.Address {
			p.issuanceModel = IssuanceEscrow
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
	if from == zeroAddr0x && p.issuanceModel == IssuanceMintOnPurchase {
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

func (p *Property) p2pTxCount() int {
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

func (p *Property) rentalStartDelay(now time.Time) (int, bool) {
	if p.Type != PropertyTypeRental || p.RentalStartExpected.Quarter < 1 {
		return 0, false
	}
	month := int(now.Month())
	actual := YearQuarter{Year: now.Year(), Quarter: (month-1)/3 + 1}
	if p.RentalStartActual != nil {
		actual = *p.RentalStartActual
	}
	return actual.ordinal() - p.RentalStartExpected.ordinal(), true
}

var AllProperties = map[string][]*Property{
	"AWWA Hotel by Ribas": {
		{
			Name:                "AWWA Hotel by Ribas B14",
			Address:             "0x216301b87404a5839bf7b8b94c646c4eb96fec79",
			Type:                PropertyTypeRental,
			AreaM2:              33,
			BinaryxPrice:        157_500,
			MarketLowValue:      2_800,
			MarketHighValue:     3_800,
			RentalStartExpected: YearQuarter{Year: 2025, Quarter: 2},
			RentalStartActual:   &YearQuarter{Year: 2025, Quarter: 4}, // first rent 01.10.2025 - 01.01.2026

		},
		{
			Name:                "AWWA Hotel by Ribas B22",
			Address:             "0xe725a80f426a7d7f5734ba69ccec507251109d09",
			Type:                PropertyTypeRental,
			AreaM2:              33,
			BinaryxPrice:        148_000, // TODO: check if this is correct
			MarketLowValue:      2_800,
			MarketHighValue:     3_800,
			RentalStartExpected: YearQuarter{Year: 2025, Quarter: 2},
			RentalStartActual:   &YearQuarter{Year: 2025, Quarter: 4}, // first rent 01.10.2025 - 01.01.2026

		},
		{
			Name:                "AWWA Hotel by Ribas A16",
			Address:             "0xdb8fc93a993e2ab0d9f7d520fd4e616cfb1d85fd",
			Type:                PropertyTypeRental,
			AreaM2:              33,
			BinaryxPrice:        157_500,
			MarketLowValue:      2_800,
			MarketHighValue:     3_800,
			RentalStartExpected: YearQuarter{Year: 2025, Quarter: 2},
			RentalStartActual:   &YearQuarter{Year: 2025, Quarter: 4}, // first rent 01.10.2025 - 01.01.2026

		},
	},
	"Aurora Villas (Bali Balance Ocean Villas)": {
		{
			Name:                "Villa Aurora 1 (Bali Balance Ocean Villa 3)",
			Address:             "0x1e3cf2eeaa6d5973e2da6fe03600ba55870dd69b",
			Type:                PropertyTypeRental,
			AreaM2:              165,
			BinaryxPrice:        428_500,
			MarketLowValue:      2_100,
			MarketHighValue:     3_200,
			RentalStartExpected: YearQuarter{Year: 2025, Quarter: 4},
		},
		{
			Name:                "Villa Aurora 2 (Bali Balance Ocean Villa 4)",
			Address:             "0x17236ed296fbd00d3dfa016879833776dd207fd6",
			Type:                PropertyTypeRental,
			AreaM2:              165,
			BinaryxPrice:        428_500,
			MarketLowValue:      2_100,
			MarketHighValue:     3_200,
			RentalStartExpected: YearQuarter{Year: 2025, Quarter: 4},
		},
	},
	"Bingin Magic Story Villas": {
		{
			Name:                "Bingin Magic Story Villa 3",
			Address:             "0xe5f846592a58bcfce912bc6fc594649397b6f519",
			Type:                PropertyTypeRental,
			AreaM2:              115,
			BinaryxPrice:        280_000,
			MarketLowValue:      2_100,
			MarketHighValue:     3_200,
			RentalStartExpected: YearQuarter{Year: 2025, Quarter: 4},
		},
	},
	"Bubbles Boutique Complex": {
		{
			Name:                "Bubbles Boutique Complex",
			Address:             "0xC1EA0Ccd94F17Ec0580DD57A34C2B521360ad4b1",
			Type:                PropertyTypeRental,
			AreaM2:              420,
			BinaryxPrice:        1_220_000,
			MarketLowValue:      2_100,
			MarketHighValue:     2_700,
			RentalStartExpected: YearQuarter{Year: 2025, Quarter: 4},
			RentalStartActual:   &YearQuarter{Year: 2025, Quarter: 4}, // first rent 01.10.25 - 31.10.25
		},
	},
	"CASCADE Villas": {
		{
			Name:                "CASCADE Villa 2",
			Address:             "0x5e55b3e941f42732f1b941f2f673dc8811355e5e",
			Type:                PropertyTypeRental,
			RentalStartExpected: YearQuarter{Year: 2026, Quarter: 1},
		},
		{
			Name:                "CASCADE Villa 3",
			Address:             "0xd5551375d5ba01ddbcb38d20ac40671f26e6ada5",
			Type:                PropertyTypeRental,
			RentalStartExpected: YearQuarter{Year: 2026, Quarter: 1},
		},
	},
	"CEMAGI Units": {
		{
			Name:                "CEMAGI Unit 3.44",
			Address:             "0x852b6995628b760c84bdd02bc143b48288d4dd3a",
			Type:                PropertyTypeConstruction,
			RentalStartExpected: YearQuarter{Year: 2026, Quarter: 2},
		},
		{
			Name:                "CEMAGI Unit 3.46",
			Address:             "0x2b7dca2c2bafdb1dac0e01068091590fbe09e478",
			Type:                PropertyTypeConstruction,
			RentalStartExpected: YearQuarter{Year: 2026, Quarter: 2},
		},
	},
	"Dukley": {
		{
			Name:                "Mountain Retreat by Dukley",
			Address:             "0x51343ee93059cbb11c4bf969a643e09117b3af6b",
			Type:                PropertyTypeRedeemed,
			BinaryxPrice:        385_000,
			RentalStartExpected: YearQuarter{Year: 2025, Quarter: 1},
		},
		{
			Name:                "Dukley Glamping 1",
			Address:             "0xad4f81d0f2f626a6ea29864f488604e6b5360e2a",
			Type:                PropertyTypeRental,
			RentalStartExpected: YearQuarter{Year: 2026, Quarter: 4},
		},
	},
	"Ecoverse Suites": {
		{
			Name:                "Ecoverse Suite",
			Address:             "0x30ed65e470be4f351abf5311769505e3f977deca",
			Type:                PropertyTypeRental,
			RentalStartExpected: YearQuarter{Year: 2025, Quarter: 3},
		},
	},
	"Hayat Green Tower": {
		{
			Name:                "Hayat Green Tower",
			Address:             "0xF9d43a9F7Fc7ee7ac47fE96De20e545243450b27",
			Type:                PropertyTypeRental,
			RentalStartExpected: YearQuarter{Year: 2023, Quarter: 3},
			RentalStartActual:   &YearQuarter{Year: 2024, Quarter: 3}, // first rent 17.08.2024 - 16.09.2024
		},
	},
	"Kammara Loft": {
		{
			Name:                "Kammara Loft",
			Address:             "0xB1B987FF1F317A47876185dE4dE9C430823Ad8c5",
			Type:                PropertyTypeRental,
			RentalStartExpected: YearQuarter{Year: 2023, Quarter: 4},
			RentalStartActual:   &YearQuarter{Year: 2024, Quarter: 4}, // first rent 10.12.2023 - 11.01.2024

		},
	},
	"Kammora Living": {
		{
			Name:                "Kammora Living",
			Address:             "0x8389AcD0e05990eF0e087A3BCed3E9C5443d0455",
			Type:                PropertyTypeRental,
			RentalStartExpected: YearQuarter{Year: 2024, Quarter: 3},
			RentalStartActual:   &YearQuarter{Year: 2024, Quarter: 3}, // first rent 01.07.2024 - 08.07.2024
		},
	},
	"Karra Loft": {
		{
			Name:                "Karra Loft 3A",
			Address:             "0x27Ceb34AC7545F78A97C0500465aCE9fA10570af",
			Type:                PropertyTypeRental,
			RentalStartExpected: YearQuarter{Year: 2023, Quarter: 3},
			RentalStartActual:   &YearQuarter{Year: 2023, Quarter: 3}, // first rent 14.08.2023 - 14.09.2023
		},
		{
			Name:                "Karra Loft 5",
			Address:             "0x2d02E704174635F5E88E17995C3a5E29f283C033",
			Type:                PropertyTypeRental,
			RentalStartExpected: YearQuarter{Year: 2023, Quarter: 3},
			RentalStartActual:   &YearQuarter{Year: 2023, Quarter: 3}, // first rent 18.09.2023 - 18.10.2023
		},
	},
	"La Casa Española Villas": {
		{
			Name:                "La Casa Española Villa 4",
			Address:             "0x7b592d8bb722324f75af834c23e6ad2058b168e1",
			Type:                PropertyTypeConstruction,
			RentalStartExpected: YearQuarter{Year: 2026, Quarter: 4},
		},
		{
			Name:                "La Casa Española Villa 6",
			Address:             "0xdd36b686a5ff910b5074e3f5483135f19e49f02c",
			Type:                PropertyTypeConstruction,
			RentalStartExpected: YearQuarter{Year: 2026, Quarter: 4},
		},
		{
			Name:                "La Casa Española Villa 8",
			Address:             "0x223270bbbe4f6dac0dc3e57d985116bdc50616ee",
			Type:                PropertyTypeConstruction,
			RentalStartExpected: YearQuarter{Year: 2026, Quarter: 4},
		},
		{
			Name:                "La Casa Española Villa 9",
			Address:             "0x89ebdfaf79308871a24c6992232984b3c84af9a8",
			Type:                PropertyTypeConstruction,
			RentalStartExpected: YearQuarter{Year: 2026, Quarter: 4},
		},
	},
	"Oasis Royal Collection": {
		{
			Name:                "Oasis Royal Collection 11a",
			Address:             "0xa26f11748ed29b3fd62e1d8e231d277a0980fb12",
			Type:                PropertyTypeRental,
			RentalStartExpected: YearQuarter{Year: 2025, Quarter: 4},
		},
		{
			Name:                "Oasis Royal Collection 18b",
			Address:             "0x1dac5a4a0e566fb2674a6b7e1cdaf2c07716eeed",
			Type:                PropertyTypeRental,
			RentalStartExpected: YearQuarter{Year: 2025, Quarter: 4},
		},
	},
	"Onyx PARQ Resort 61": {
		{
			Name:                "Onyx PARQ Resort 61",
			Address:             "0xA07DB641FC95067a2Fe68b6224a9dD39564bFd57",
			Type:                PropertyTypeRental,
			RentalStartExpected: YearQuarter{Year: 2023, Quarter: 2},
			RentalStartActual:   &YearQuarter{Year: 2023, Quarter: 2}, // first rent 01.04.2023 - 01.05.2023
		},
	},
	"Roots Villas": {
		{
			Name:                "Roots Villa 1",
			Address:             "0xbde380b4cc582d440255ebd89ff1839dcfad5d7b",
			Type:                PropertyTypeConstruction,
			RentalStartExpected: YearQuarter{Year: 2026, Quarter: 3},
		},
		{
			Name:                "Roots Villa 3",
			Address:             "0xc0a4b2e29bd44d3b798a02edc039711f03572739",
			Type:                PropertyTypeConstruction,
			RentalStartExpected: YearQuarter{Year: 2026, Quarter: 3},
		},
		{
			Name:                "Roots Villa 4",
			Address:             "0xb2b9f922c0494dbf08636b1dbcf6fcba0878a605",
			Type:                PropertyTypeConstruction,
			RentalStartExpected: YearQuarter{Year: 2026, Quarter: 3},
		},
		{
			Name:                "Roots Villa 5",
			Address:             "0x0ef68e86c3c9bc6187c69770053919e6b35991f6",
			Type:                PropertyTypeConstruction,
			RentalStartExpected: YearQuarter{Year: 2026, Quarter: 3},
		},
	},
	"Taryan Dragon Jungle View": {
		{
			Name:                "Taryan Dragon Jungle View",
			Address:             "0x4bd4d7003a6ce76b9ad3ee364a29801c170b1ff5",
			Type:                PropertyTypeConstruction,
			RentalStartExpected: YearQuarter{Year: 2027, Quarter: 4},
		},
	},
	"Tropical Loft Villas": {
		{
			Name:                "Tropical Loft Villa 2",
			Address:             "0x4b17845F255cC19dB2612ab8577Ea1e0852BBBd7",
			Type:                PropertyTypeRental,
			RentalStartExpected: YearQuarter{Year: 2025, Quarter: 1},
			RentalStartActual:   &YearQuarter{Year: 2025, Quarter: 1}, // first rent 28.03.25 - 28.04.25
		},
		{
			Name:                "Tropical Loft Villa 3",
			Address:             "0x56467B7E0cF2116A1C7664eE60db77ED24709293",
			Type:                PropertyTypeRental,
			RentalStartExpected: YearQuarter{Year: 2025, Quarter: 2},
			RentalStartActual:   &YearQuarter{Year: 2025, Quarter: 2}, // first rent 01.05.25 - 01.06.25
		},
		{
			Name:                "Tropical Loft Villa 4",
			Address:             "0x6a6F5681678Cc599d4d4Ae55270070406561DCa7",
			Type:                PropertyTypeRental,
			RentalStartExpected: YearQuarter{Year: 2025, Quarter: 2},
			RentalStartActual:   &YearQuarter{Year: 2025, Quarter: 2}, // first rent 20.06.25 - 20.07.25
		},
	},
	"Vesna Townhouse": {
		{
			Name:                "Vesna Townhouse",
			Address:             "0x09558935e9cA1c4985F96163b62Fd850616F6e33",
			Type:                PropertyTypeRental,
			RentalStartExpected: YearQuarter{Year: 2024, Quarter: 2},
			RentalStartActual:   &YearQuarter{Year: 2024, Quarter: 2}, // first rent 05.04.24 - 27.04.24
		},
	},
}
