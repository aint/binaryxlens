package binaryx

import (
	"fmt"
	"time"
)

const (
	Fair        ValueStatus = "fair"
	Rich        ValueStatus = "rich"
	Overpriced  ValueStatus = "overpriced"
	Underpriced ValueStatus = "underpriced"

	Bali       Location = "bali"
	Turkey     Location = "turkey"
	Montenegro Location = "montenegro"

	Construction RealEstateState = "construction"
	Rental       RealEstateState = "rental"
	Redeemed     RealEstateState = "redeemed"

	MintOnPurchase IssuanceModel = 1 // 0x0 → buyer on each sale
	Escrow         IssuanceModel = 2 // contract → buyer; 0x0 → contract is inventory
)

type (
	ValueStatus     string
	Location        string
	RealEstateState string

	// IssuanceModel is how initial sales leave issuer control.
	IssuanceModel int
)

type Value struct {
	Price  uint
	Status ValueStatus
}

type YearQuarter struct {
	Year    int
	Quarter int // 1..4
}

func NewValue(price uint, deltaFn func(uint) (float64, bool)) Value {
	v := Value{Price: price}
	delta, ok := deltaFn(price)
	if !ok {
		return v
	}
	switch {
	case delta < -0.10:
		v.Status = Underpriced
	case delta <= 0.10:
		v.Status = Fair
	case delta <= 0.20:
		v.Status = Rich
	default:
		v.Status = Overpriced
	}
	return v
}

func NewYearQuarter(t time.Time) YearQuarter {
	month := int(t.Month())
	return YearQuarter{Year: t.Year(), Quarter: (month-1)/3 + 1}
}

func (yq YearQuarter) String() string {
	return fmt.Sprintf("%d Q%d", yq.Year, yq.Quarter)
}

func (yq YearQuarter) QuartersSince(start YearQuarter) int {
	return yq.ordinal() - start.ordinal()
}

func (yq YearQuarter) ordinal() int {
	return yq.Year*4 + yq.Quarter - 1
}

func (l Location) Flag() string {
	switch l {
	case Bali:
		return "🇮🇩"
	case Turkey:
		return "🇹🇷"
	case Montenegro:
		return "🇲🇪"
	default:
		return ""
	}
}
