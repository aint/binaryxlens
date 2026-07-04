package internal

import (
	"fmt"
	"math/big"
	"strings"
)

const timeDateOnly = "2006-01-02"

func FormatBigInt(raw *big.Int, decimals uint8) string {
	if raw == nil || raw.Sign() == 0 {
		return "0"
	}
	if decimals == 0 {
		return raw.String()
	}
	denom := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	ip := new(big.Int).Quo(raw, denom)
	fp := new(big.Int).Mod(new(big.Int).Set(raw), denom)
	if fp.Sign() == 0 {
		return ip.String()
	}
	frac := fp.Text(10)
	for len(frac) < int(decimals) {
		frac = "0" + frac
	}
	frac = strings.TrimRight(frac, "0")
	return ip.String() + "." + frac
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