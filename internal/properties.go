package internal

import (
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aint/binaryxlens/internal/polygonscan"
)

// IssuanceModel is how initial sales leave issuer control.
type IssuanceModel int

const (
	IssuanceMintOnPurchase IssuanceModel = iota // 0x0 → buyer on each sale
	IssuanceEscrow                              // contract → buyer; 0x0 → contract is inventory
)

type Property struct {
	Contract
	txs                    []polygonscan.TokenTransfer
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

type Contract struct {
	Name     string
	Address  string
	ExitDate YearQuarter
	Redeemed bool
}

type YearQuarter struct {
	Year    int
	Quarter int // 1..4
}

func (yq YearQuarter) String() string {
	return fmt.Sprintf("%d Q%d", yq.Year, yq.Quarter)
}

func NewProperty(contract Contract, client *polygonscan.Client, scanPause time.Duration) (*Property, error) {
	contract.Address = strings.ToLower(contract.Address)
	property := &Property{
		Contract: contract,
	}

	var err error
	property.txs, err = client.FetchAllTokenTx(property.Address, 1000, scanPause)
	if err != nil {
		return nil, fmt.Errorf("fetch all token tx: %v", err)
	}
	if len(property.txs) == 0 {
		// TODO: mark as new one instead of returning error
		return nil, fmt.Errorf("no transactions found")
	}
	property.resolveIssuanceModel()

	err = property.extractDecimal()
	if err != nil {
		return nil, fmt.Errorf("extract decimal: %v", err)
	}

	property.TotalSupplyRaw, err = client.GetTotalSupply(property.Address)
	if err != nil {
		return nil, fmt.Errorf("get total supply: %v", err)
	}
	if property.Redeemed {
		property.TotalSupplyRaw, err = calculateIssuedSupply(property.txs)
		if err != nil {
			return nil, fmt.Errorf("issued supply: %v", err)
		}
	}
	if property.TotalSupplyRaw.Sign() == 0 {
		return nil, errors.New("total supply is zero")
	}

	property.calculateBoughtRaw()
	property.RemainingRaw = new(big.Int).Sub(property.TotalSupplyRaw, property.BoughtRaw)

	err = property.buildHolders()
	if err != nil {
		return nil, fmt.Errorf("build holders: %v", err)
	}

	err = property.buildInitialSaleDailySeries()
	if err != nil {
		return nil, fmt.Errorf("build initial sale daily series: %v", err)
	}

	err = property.buildP2PSaleWeeklySeries()
	if err != nil {
		return nil, fmt.Errorf("build p2p sale weekly series: %v", err)
	}

	err = property.calculateMovingAverageETA()
	if err != nil {
		return nil, fmt.Errorf("calculate ETAs: %v", err)
	}

	fmt.Printf("Property %q initialized\n", property.Name)

	return property, nil
}

func (p *Property) resolveIssuanceModel() {
	p.issuanceModel = IssuanceMintOnPurchase
	for _, tx := range p.txs {
		if tx.From == p.Address {
			p.issuanceModel = IssuanceEscrow
			return
		}
	}
}

func (p *Property) calculateBoughtRaw() {
	boughtAmount := big.NewInt(0)
	for _, tx := range p.txs {
		v, ok := new(big.Int).SetString(tx.Value, 10)
		if !ok {
			fmt.Fprintf(os.Stderr, "parse value %q\n", tx.Value)
			continue
		}

		if p.isInitialSale(tx.From) {
			boughtAmount.Add(boughtAmount, v)
		}
	}

	p.BoughtRaw = boughtAmount
}

func calculateIssuedSupply(txs []polygonscan.TokenTransfer) (*big.Int, error) {
	supply, maxSupply := big.NewInt(0), big.NewInt(0)
	for _, tx := range txs {
		v, ok := new(big.Int).SetString(tx.Value, 10)
		if !ok {
			return nil, fmt.Errorf("parse value %q", tx.Value)
		}
		if tx.From == zeroAddr0x {
			supply.Add(supply, v)
		}
		if tx.To == zeroAddr0x {
			supply.Sub(supply, v)
		}
		if supply.Cmp(maxSupply) > 0 {
			maxSupply.Set(supply)
		}
	}
	return maxSupply, nil
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
	for _, tx := range p.txs {
		if p.isP2PTransfer(tx.From, tx.To) {
			count++
		}
	}
	return count
}

func (p *Property) extractDecimal() error {
	decimalStr := strings.TrimSpace(p.txs[0].TokenDecimal)
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

var AllPropertyContracts = map[string][]Contract{
	"Dukley": {
		{
			Name:     "Mountain Retreat by Dukley",
			Address:  "0x51343ee93059cbb11c4bf969a643e09117b3af6b",
			ExitDate: YearQuarter{Year: 2024, Quarter: 4},
			Redeemed: true,
		},
		{
			Name:     "Dukley Glamping 1",
			Address:  "0xad4f81d0f2f626a6ea29864f488604e6b5360e2a",
			ExitDate: YearQuarter{Year: 2026, Quarter: 4},
		},
	},
	"Bubbles Boutique Complex": {
		{
			Name:     "Bubbles Boutique Complex",
			Address:  "0xC1EA0Ccd94F17Ec0580DD57A34C2B521360ad4b1",
			ExitDate: YearQuarter{Year: 2050, Quarter: 4},
		},
	},
	"Ecoverse Suites": {
		{
			Name:     "Ecoverse Suite",
			Address:  "0x30ed65e470be4f351abf5311769505e3f977deca",
			ExitDate: YearQuarter{Year: 2026, Quarter: 2},
		},
	},
	"Taryan Dragon Jungle Views": {
		{
			Name:     "Taryan Dragon Jungle View",
			Address:  "0x4bd4d7003a6ce76b9ad3ee364a29801c170b1ff5",
			ExitDate: YearQuarter{Year: 2027, Quarter: 4},
		},
	},
	"Bingin Magic Story Villas": {
		{
			Name:     "Bingin Magic Story Villa 3",
			Address:  "0xe5f846592a58bcfce912bc6fc594649397b6f519",
			ExitDate: YearQuarter{Year: 2026, Quarter: 2},
		},
	},
	"CEMAGI Units": {
		{
			Name:     "CEMAGI Unit 3.44",
			Address:  "0x852b6995628b760c84bdd02bc143b48288d4dd3a",
			ExitDate: YearQuarter{Year: 2026, Quarter: 2},
		},
		{
			Name:     "CEMAGI Unit 3.46",
			Address:  "0x2b7dca2c2bafdb1dac0e01068091590fbe09e478",
			ExitDate: YearQuarter{Year: 2026, Quarter: 2},
		},
	},
	"CASCADE Villas": {
		{
			Name:     "CASCADE Villa 2",
			Address:  "0x5e55b3e941f42732f1b941f2f673dc8811355e5e",
			ExitDate: YearQuarter{Year: 2026, Quarter: 2},
		},
		{
			Name:     "CASCADE Villa 3",
			Address:  "0xd5551375d5ba01ddbcb38d20ac40671f26e6ada5",
			ExitDate: YearQuarter{Year: 2026, Quarter: 2},
		},
	},
	"Bali Balance Ocean Villas": {
		{
			Name:     "Bali Balance Ocean Villa 3",
			Address:  "0x1e3cf2eeaa6d5973e2da6fe03600ba55870dd69b",
			ExitDate: YearQuarter{Year: 2026, Quarter: 2},
		},
		{
			Name:     "Bali Balance Ocean Villa 4",
			Address:  "0x17236ed296fbd00d3dfa016879833776dd207fd6",
			ExitDate: YearQuarter{Year: 2026, Quarter: 2},
		},
	},
	"Oasis Royal Collection": {
		{
			Name:     "Oasis Royal Collection 11a",
			Address:  "0xa26f11748ed29b3fd62e1d8e231d277a0980fb12",
			ExitDate: YearQuarter{Year: 2025, Quarter: 4},
		},
		{
			Name:     "Oasis Royal Collection 18b",
			Address:  "0x1dac5a4a0e566fb2674a6b7e1cdaf2c07716eeed",
			ExitDate: YearQuarter{Year: 2025, Quarter: 4},
		},
	},
	"AWWA Hotel by Ribas": {
		{
			Name:     "AWWA Hotel by Ribas B14",
			Address:  "0x216301b87404a5839bf7b8b94c646c4eb96fec79",
			ExitDate: YearQuarter{Year: 2025, Quarter: 2},
		},
		{
			Name:     "AWWA Hotel by Ribas B22",
			Address:  "0xe725a80f426a7d7f5734ba69ccec507251109d09",
			ExitDate: YearQuarter{Year: 2025, Quarter: 2},
		},
		{
			Name:     "AWWA Hotel by Ribas A16",
			Address:  "0xdb8fc93a993e2ab0d9f7d520fd4e616cfb1d85fd",
			ExitDate: YearQuarter{Year: 2025, Quarter: 2},
		},
	},
	"Roots Villas": {
		{
			Name:     "Roots Villa 1",
			Address:  "0xbde380b4cc582d440255ebd89ff1839dcfad5d7b",
			ExitDate: YearQuarter{Year: 2026, Quarter: 3},
		},
		{
			Name:     "Roots Villa 3",
			Address:  "0xc0a4b2e29bd44d3b798a02edc039711f03572739",
			ExitDate: YearQuarter{Year: 2026, Quarter: 3},
		},
		{
			Name:     "Roots Villa 4",
			Address:  "0xb2b9f922c0494dbf08636b1dbcf6fcba0878a605",
			ExitDate: YearQuarter{Year: 2026, Quarter: 3},
		},
		{
			Name:     "Roots Villa 5",
			Address:  "0x0ef68e86c3c9bc6187c69770053919e6b35991f6",
			ExitDate: YearQuarter{Year: 2026, Quarter: 3},
		},
	},
	"La Casa Española Villas": {
		{
			Name:     "La Casa Española Villa 4",
			Address:  "0x7b592d8bb722324f75af834c23e6ad2058b168e1",
			ExitDate: YearQuarter{Year: 2026, Quarter: 4},
		},
		{
			Name:     "La Casa Española Villa 6",
			Address:  "0xdd36b686a5ff910b5074e3f5483135f19e49f02c",
			ExitDate: YearQuarter{Year: 2026, Quarter: 4},
		},
		{
			Name:     "La Casa Española Villa 8",
			Address:  "0x223270bbbe4f6dac0dc3e57d985116bdc50616ee",
			ExitDate: YearQuarter{Year: 2026, Quarter: 4},
		},
		{
			Name:     "La Casa Española Villa 9",
			Address:  "0x89ebdfaf79308871a24c6992232984b3c84af9a8",
			ExitDate: YearQuarter{Year: 2026, Quarter: 4},
		},
	},
}
