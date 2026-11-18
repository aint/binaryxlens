package site

import (
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/aint/binaryxlens/internal/binaryx"
)

func TestTrendsAggregatesHolderActivityByBalance(t *testing.T) {
	today := time.Time(binaryx.NewDay(time.Now().UTC()))
	holder := func(balance int64, recent, previous, older int64) globalHolder {
		return globalHolder{
			Address:   "holder",
			TotalUSDT: big.NewInt(balance),
			DailyActivity: binaryx.DailyActivity{
				binaryx.NewDay(today.AddDate(0, 0, -15)): {UsdtIn: big.NewInt(recent), UsdtOut: big.NewInt(0)},
				binaryx.NewDay(today.AddDate(0, 0, -45)): {UsdtIn: big.NewInt(previous), UsdtOut: big.NewInt(0)},
				binaryx.NewDay(today.AddDate(0, 0, -75)): {UsdtIn: big.NewInt(older), UsdtOut: big.NewInt(0)},
			},
		}
	}

	holders := []globalHolder{
		holder(150_000_000000, 100, 200, 300),
		holder(50_000_000000, 10, 20, 30),
		holder(5_000_000000, 1, 2, 3),
		holder(4_999_000000, 1_000, 2_000, 3_000),
		holder(4_998_000000, 10, 20, 30),
	}

	want := []trend{
		{Name: "whale", DeltasUSDT: []*big.Int{big.NewInt(200), big.NewInt(300), big.NewInt(100)}},
		{Name: "shark", DeltasUSDT: []*big.Int{big.NewInt(20), big.NewInt(30), big.NewInt(10)}},
		{Name: "dolphin", DeltasUSDT: []*big.Int{big.NewInt(0), big.NewInt(0), big.NewInt(0)}},
		{Name: "fish", DeltasUSDT: []*big.Int{big.NewInt(0), big.NewInt(0), big.NewInt(0)}},
		{Name: "crab", DeltasUSDT: []*big.Int{big.NewInt(2), big.NewInt(3), big.NewInt(1)}},
		{Name: "shrimp", DeltasUSDT: []*big.Int{big.NewInt(2_020), big.NewInt(3_030), big.NewInt(1_010)}},
	}

	if got := trends(holders); !reflect.DeepEqual(got, want) {
		t.Errorf("\ngot: %v,\nwant: %v", got, want)
	}
}

func TestTrendsIncludesCEOActivity(t *testing.T) {
	today := time.Time(binaryx.NewDay(time.Now().UTC()))
	activity := binaryx.DailyActivity{
		binaryx.NewDay(today.AddDate(0, 0, -15)): {UsdtIn: big.NewInt(7), UsdtOut: big.NewInt(0)},
		binaryx.NewDay(today.AddDate(0, 0, -45)): {UsdtIn: big.NewInt(8), UsdtOut: big.NewInt(0)},
		binaryx.NewDay(today.AddDate(0, 0, -75)): {UsdtIn: big.NewInt(9), UsdtOut: big.NewInt(0)},
	}

	trends := trends([]globalHolder{{Address: ceoWallet, TotalUSDT: big.NewInt(0), DailyActivity: activity}})
	if len(trends) != len(trendBuckets)+1 {
		t.Fatalf("trends() returned %d entries, want %d", len(trends), len(trendBuckets)+1)
	}

	gotCEOTrend := trends[len(trends)-1]
	wantCEOTrend := trend{Name: "ceo", DeltasUSDT: []*big.Int{big.NewInt(8), big.NewInt(9), big.NewInt(7)}}
	if !reflect.DeepEqual(gotCEOTrend, wantCEOTrend) {
		t.Errorf("CEO trend: %v, want: %v", gotCEOTrend, wantCEOTrend)
	}

	gotShrimpTrend := trends[len(trends)-2]
	wantShrimpTrend := trend{Name: "shrimp", DeltasUSDT: []*big.Int{big.NewInt(8), big.NewInt(9), big.NewInt(7)}}
	if !reflect.DeepEqual(gotShrimpTrend, wantShrimpTrend) {
		t.Errorf("Shrimp trend: %v, want: %v", gotShrimpTrend, wantShrimpTrend)
	}
}
