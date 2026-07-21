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
	txs            []polygonscan.TokenTransfer
	issuanceModel  IssuanceModel
	DailyPoints    []DailyPoint
	ETAs           []ETA
	Holders        []Holder
	TotalSupplyRaw *big.Int
	BoughtRaw      *big.Int
	RemainingRaw   *big.Int
	Decimal        uint8
}

type Contract struct {
	Name     string
	Address  string
	ExitDate YearQuarter
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

	property.calculateBoughtRaw()
	property.RemainingRaw = new(big.Int).Sub(property.TotalSupplyRaw, property.BoughtRaw)

	err = property.buildHolders()
	if err != nil {
		return nil, fmt.Errorf("build holders: %v", err)
	}

	err = property.buildDailySeries()
	if err != nil {
		return nil, fmt.Errorf("build daily series: %v", err)
	}

	err = property.calculateMovingAverageETA()
	if err != nil {
		return nil, fmt.Errorf("calculate ETAs: %v", err)
	}

	// if 100% bought, replace ETA with start - end period and token bought rate

	fmt.Printf("Property '%s' initialized\n", property.Name)

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

var LaCasaEspanolaVilla4 = Contract{
	Name:     "La Casa Española Villa 4",
	Address:  "0x7b592d8bb722324f75af834c23e6ad2058b168e1",
	ExitDate: YearQuarter{Year: 2026, Quarter: 4},
}
var LaCasaEspanolaVilla6 = Contract{
	Name:     "La Casa Española Villa 6",
	Address:  "0xdd36b686a5ff910b5074e3f5483135f19e49f02c",
	ExitDate: YearQuarter{Year: 2026, Quarter: 4},
}
var LaCasaEspanolaVilla8 = Contract{
	Name:     "La Casa Española Villa 8",
	Address:  "0x223270bbbe4f6dac0dc3e57d985116bdc50616ee",
	ExitDate: YearQuarter{Year: 2026, Quarter: 4},
}
var LaCasaEspanolaVilla9 = Contract{
	Name:     "La Casa Española Villa 9",
	Address:  "0x89ebdfaf79308871a24c6992232984b3c84af9a8",
	ExitDate: YearQuarter{Year: 2026, Quarter: 4},
}

var LaCasaEspanolaVillas = []Contract{
	LaCasaEspanolaVilla4,
	LaCasaEspanolaVilla6,
	LaCasaEspanolaVilla8,
	LaCasaEspanolaVilla9,
}

var RootsVilla1 = Contract{
	Name:     "Roots Villa 1",
	Address:  "0xbde380b4cc582d440255ebd89ff1839dcfad5d7b",
	ExitDate: YearQuarter{Year: 2026, Quarter: 3},
}
var RootsVilla3 = Contract{
	Name:     "Roots Villa 3",
	Address:  "0xc0a4b2e29bd44d3b798a02edc039711f03572739",
	ExitDate: YearQuarter{Year: 2026, Quarter: 3},
}
var RootsVilla4 = Contract{
	Name:     "Roots Villa 4",
	Address:  "0xb2b9f922c0494dbf08636b1dbcf6fcba0878a605",
	ExitDate: YearQuarter{Year: 2026, Quarter: 3},
}
var RootsVilla5 = Contract{
	Name:     "Roots Villa 5",
	Address:  "0x0ef68e86c3c9bc6187c69770053919e6b35991f6",
	ExitDate: YearQuarter{Year: 2026, Quarter: 3},
}

var RootsVillas = []Contract{
	RootsVilla1,
	RootsVilla3,
	RootsVilla4,
	RootsVilla5,
}

var DukleyGlamping1 = Contract{
	Name:     "Dukley Glamping 1",
	Address:  "0xad4f81d0f2f626a6ea29864f488604e6b5360e2a",
	ExitDate: YearQuarter{Year: 2026, Quarter: 4},
}
var MountainRetreatByDukley = Contract{
	Name:     "Mountain Retreat by Dukley",
	Address:  "0x51343ee93059cbb11c4bf969a643e09117b3af6b",
	ExitDate: YearQuarter{Year: 2024, Quarter: 4},
}

var Dukley = []Contract{
	DukleyGlamping1,
	MountainRetreatByDukley,
}

var CemagiUnit344 = Contract{
	Name:     "CEMAGI Unit 3.44",
	Address:  "0x852b6995628b760c84bdd02bc143b48288d4dd3a",
	ExitDate: YearQuarter{Year: 2026, Quarter: 2},
}
var CemagiUnit346 = Contract{
	Name:     "CEMAGI Unit 3.46",
	Address:  "0x2b7dca2c2bafdb1dac0e01068091590fbe09e478",
	ExitDate: YearQuarter{Year: 2026, Quarter: 2},
}

var CemagiUnits = []Contract{
	CemagiUnit344,
	CemagiUnit346,
}

var CadecasVilla2 = Contract{
	Name:     "CASCADE Villa 2",
	Address:  "0x5e55b3e941f42732f1b941f2f673dc8811355e5e",
	ExitDate: YearQuarter{Year: 2026, Quarter: 2},
}
var CadecasVilla3 = Contract{
	Name:     "CASCADE Villa 3",
	Address:  "0xd5551375d5ba01ddbcb38d20ac40671f26e6ada5",
	ExitDate: YearQuarter{Year: 2026, Quarter: 2},
}

var CadecasVillas = []Contract{
	CadecasVilla2,
	CadecasVilla3,
}

var BaliBalanceOceanVilla3 = Contract{
	Name:     "Bali Balance Ocean Villa 3",
	Address:  "0x1e3cf2eeaa6d5973e2da6fe03600ba55870dd69b",
	ExitDate: YearQuarter{Year: 2026, Quarter: 2},
}
var BaliBalanceOceanVilla4 = Contract{
	Name:     "Bali Balance Ocean Villa 4",
	Address:  "0x17236ed296fbd00d3dfa016879833776dd207fd6",
	ExitDate: YearQuarter{Year: 2026, Quarter: 2},
}

var BaliBalanceOceanVillas = []Contract{
	BaliBalanceOceanVilla3,
	BaliBalanceOceanVilla4,
}

var BinginMagicStoryVilla3 = Contract{
	Name:     "Bingin Magic Story Villa 3",
	Address:  "0xe5f846592a58bcfce912bc6fc594649397b6f519",
	ExitDate: YearQuarter{Year: 2026, Quarter: 2},
}

var BinginMagicStoryVillas = []Contract{
	BinginMagicStoryVilla3,
}

var OasisRoyalCollection11a = Contract{
	Name:     "Oasis Royal Collection 11a",
	Address:  "0xa26f11748ed29b3fd62e1d8e231d277a0980fb12",
	ExitDate: YearQuarter{Year: 2025, Quarter: 4},
}
var OasisRoyalCollection18b = Contract{
	Name:     "Oasis Royal Collection 18b",
	Address:  "0x1dac5a4a0e566fb2674a6b7e1cdaf2c07716eeed",
	ExitDate: YearQuarter{Year: 2025, Quarter: 4},
}

var OasisRoyalCollection = []Contract{
	OasisRoyalCollection11a,
	OasisRoyalCollection18b,
}

var TaryanDragonJungleView = Contract{
	Name:     "Taryan Dragon Jungle View",
	Address:  "0x4bd4d7003a6ce76b9ad3ee364a29801c170b1ff5",
	ExitDate: YearQuarter{Year: 2027, Quarter: 4},
}

var TaryanDragonJungleViews = []Contract{
	TaryanDragonJungleView,
}

var AWWAHotelByRibasB14 = Contract{
	Name:     "AWWA Hotel by Ribas B14",
	Address:  "0x216301b87404a5839bf7b8b94c646c4eb96fec79",
	ExitDate: YearQuarter{Year: 2025, Quarter: 2},
}
var AWWAHotelByRibasB22 = Contract{
	Name:     "AWWA Hotel by Ribas B22",
	Address:  "0xe725a80f426a7d7f5734ba69ccec507251109d09",
	ExitDate: YearQuarter{Year: 2025, Quarter: 2},
}
var AWWAHotelByRibasA16 = Contract{
	Name:     "AWWA Hotel by Ribas A16",
	Address:  "0xdb8fc93a993e2ab0d9f7d520fd4e616cfb1d85fd",
	ExitDate: YearQuarter{Year: 2025, Quarter: 2},
}

var AWWAHotelByRibas = []Contract{
	AWWAHotelByRibasB14,
	AWWAHotelByRibasB22,
	AWWAHotelByRibasA16,
}

var EcoverseSuite = Contract{
	Name:     "Ecoverse Suite",
	Address:  "0x30ed65e470be4f351abf5311769505e3f977deca",
	ExitDate: YearQuarter{Year: 2026, Quarter: 2},
}

var EcoverseSuites = []Contract{
	EcoverseSuite,
}

var AllTokenDetails = [][]Contract{
	LaCasaEspanolaVillas,
	RootsVillas,
	Dukley,
	CemagiUnits,
	CadecasVillas,
	BaliBalanceOceanVillas,
	BinginMagicStoryVillas,
	OasisRoyalCollection,
	TaryanDragonJungleViews,
	AWWAHotelByRibas,
	EcoverseSuites,
}
