package site

import (
	_ "embed"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/aint/binaryxlens/internal/binaryx"
	"github.com/aint/binaryxlens/internal/polygonscan"
)

//go:embed trends.html
var trendsTemplateSrc []byte

var trendsPlaceholder = []byte("__TRENDS__")

var trendBuckets = []struct {
	name string
	min  *big.Int
}{
	{"100k+", big.NewInt(100_000_000000)},
	{"50k-100k", big.NewInt(50_000_000000)},
	{"20k-50k", big.NewInt(20_000_000000)},
	{"5k-20k", big.NewInt(5_000_000000)},
	{"5k-0", big.NewInt(0)},
}

type trend struct {
	Name           string
	MonthDeltaUSDT *big.Int
}

func WriteTrends(path string, updated time.Time, projects []*binaryx.Project) error {
	payload := []byte(trendHTML(projects))
	return write(path, updated, trendsTemplateSrc, trendsPlaceholder, payload)
}

func trendHTML(projects []*binaryx.Project) string {
	var b strings.Builder
	for _, t := range trends(globalHolders(projects)) {
		value, class := "—", "flat"
		if t.MonthDeltaUSDT != nil && t.MonthDeltaUSDT.Sign() != 0 {
			sign := "+"
			class = "up"
			if t.MonthDeltaUSDT.Sign() < 0 {
				sign, class = "-", "down"
			}
			value = sign + binaryx.FormatBigInt(new(big.Int).Abs(t.MonthDeltaUSDT), polygonscan.USDTDecimal) + " USDT"
		}
		fmt.Fprintf(&b, `
				<div class="trend-row">
					<span class="item"><span class="name" data-i18n="trends.type.%s">%s</span></span>
					<span class="delta %s">%s</span>
				</div>`,
			t.Name,
			t.Name,
			class,
			value,
		)
	}
	return b.String()
}

func trends(holders []globalHolder) []trend {
	trends := make([]trend, len(trendBuckets))
	for i, bucket := range trendBuckets {
		trends[i] = trend{
			Name:           bucket.name,
			MonthDeltaUSDT: new(big.Int),
		}
	}

	for _, h := range holders {
		for i, bucket := range trendBuckets {
			if h.TotalUSDT.Cmp(bucket.min) >= 0 {
				trends[i].MonthDeltaUSDT.Add(trends[i].MonthDeltaUSDT, h.MonthDeltaUSDT)
				break
			}
		}
	}

	return trends
}
