package internal

import (
	"fmt"
	"math"
	"math/big"
)

const timeDateOnly = "2006-01-02"

// FormatBigInt prints a token or USDT amount with two decimal places.
// Digits past that round half away from zero: 1.225 becomes 1.23, 1.224 becomes 1.22.
func FormatBigInt(raw *big.Int, decimals uint8) string {
	if raw == nil || raw.Sign() == 0 {
		return "0.00"
	}
	v := new(big.Int).Abs(raw)
	const places uint8 = 2
	if decimals > places {
		shift := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals-places)), nil)
		quo, rem := new(big.Int).QuoRem(v, shift, new(big.Int))
		if rem.Cmp(new(big.Int).Rsh(new(big.Int).Set(shift), 1)) >= 0 {
			quo.Add(quo, big.NewInt(1))
		}
		v = quo
	} else if decimals < places {
		shift := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(places-decimals)), nil)
		v.Mul(v, shift)
	}
	s := formatFixedPlaces(v, places)
	if raw.Sign() < 0 {
		return "-" + s
	}
	return s
}

func formatFixedPlaces(v *big.Int, decimals uint8) string {
	denom := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	ip := new(big.Int).Quo(v, denom)
	fp := new(big.Int).Mod(v, denom)
	frac := fp.Text(10)
	for len(frac) < int(decimals) {
		frac = "0" + frac
	}
	return ip.String() + "." + frac
}

func formatDelta(raw *big.Int, decimals uint8) string {
	if raw == nil || raw.Sign() == 0 {
		return "0.00"
	}
	s := FormatBigInt(new(big.Int).Abs(raw), decimals)
	if s == "0.00" {
		return "0.00"
	}
	if raw.Sign() < 0 {
		return "-" + s
	}
	return "+" + s
}

func bigIntToFloat(raw *big.Int, decimals uint8) float64 {
	if raw == nil || raw.Sign() == 0 {
		return 0
	}
	denom := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	r := new(big.Rat).SetFrac(new(big.Int).Set(raw), denom)
	f, _ := r.Float64()
	return math.Round(f*100) / 100
}

func PercentOf(part, whole *big.Int) string {
	return fmt.Sprintf("%.2f", PercentFloat(part, whole))
}

func PercentFloat(part, whole *big.Int) float64 {
	if whole == nil || whole.Sign() == 0 || part == nil {
		return 0
	}
	r := new(big.Rat).SetFrac(part, whole)
	r.Mul(r, big.NewRat(100, 1))
	f, _ := r.Float64()
	return f
}

func FormatBigRat(rawAmount *big.Rat, decimals uint8, prec int) string {
	if rawAmount == nil || rawAmount.Sign() == 0 {
		return "0"
	}
	scale := big.NewInt(1)
	if decimals > 0 {
		scale = new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	}
	human := new(big.Rat).Quo(rawAmount, new(big.Rat).SetInt(scale))
	return human.FloatString(prec)
}

func negBigInt(v *big.Int) *big.Int {
	if v == nil || v.Sign() == 0 {
		return nil
	}
	return new(big.Int).Neg(v)
}
