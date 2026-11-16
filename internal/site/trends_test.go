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
		{Name: "150k+", DeltasUSDT: []*big.Int{big.NewInt(200), big.NewInt(300), big.NewInt(100)}},
		{Name: "50k-150k", DeltasUSDT: []*big.Int{big.NewInt(20), big.NewInt(30), big.NewInt(10)}},
		{Name: "5k-50k", DeltasUSDT: []*big.Int{big.NewInt(2), big.NewInt(3), big.NewInt(1)}},
		{Name: "5k-0", DeltasUSDT: []*big.Int{big.NewInt(2_020), big.NewInt(3_030), big.NewInt(1_010)}},
	}

	if got := trends(holders); !reflect.DeepEqual(got, want) {
		t.Errorf("trends() = %v, want %v", got, want)
	}
}

func TestTrendsIncludesCEOActivity(t *testing.T) {
	today := time.Time(binaryx.NewDay(time.Now().UTC()))
	activity := binaryx.DailyActivity{
		binaryx.NewDay(today.AddDate(0, 0, -15)): {UsdtIn: big.NewInt(7), UsdtOut: big.NewInt(0)},
		binaryx.NewDay(today.AddDate(0, 0, -45)): {UsdtIn: big.NewInt(8), UsdtOut: big.NewInt(0)},
		binaryx.NewDay(today.AddDate(0, 0, -75)): {UsdtIn: big.NewInt(9), UsdtOut: big.NewInt(0)},
	}

	got := trends([]globalHolder{{Address: ceoWallet, TotalUSDT: big.NewInt(0), DailyActivity: activity}})
	if len(got) != len(trendBuckets)+1 {
		t.Fatalf("trends() returned %d entries, want %d", len(got), len(trendBuckets)+1)
	}
	want := trend{Name: "ceo", DeltasUSDT: []*big.Int{big.NewInt(8), big.NewInt(9), big.NewInt(7)}}
	if !reflect.DeepEqual(got[len(got)-1], want) {
		t.Errorf("CEO trend = %v, want %v", got[len(got)-1], want)
	}
	if got[3].DeltasUSDT[0].Cmp(big.NewInt(8)) != 0 || got[3].DeltasUSDT[1].Cmp(big.NewInt(9)) != 0 || got[3].DeltasUSDT[2].Cmp(big.NewInt(7)) != 0 {
		t.Errorf("CEO holder should also contribute to the matching balance group: %v", got[3].DeltasUSDT)
	}
}
