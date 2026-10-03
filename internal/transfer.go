package internal

import (
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/aint/binaryxlens/internal/polygonscan"
)

type transfer struct {
	Hash  string
	From  string
	To    string
	Value *big.Int
	Time  time.Time
	USDT  *big.Int
}

func newTransfers(tokenTransfers []polygonscan.TokenTransfer) ([]transfer, error) {
	transfers := make([]transfer, 0, len(tokenTransfers))
	for _, tt := range tokenTransfers {
		v, ok := new(big.Int).SetString(tt.Value, 10)
		if !ok {
			return nil, fmt.Errorf("parse value %q", tt.Value)
		}
		ts, err := strconv.ParseInt(tt.TimeStamp, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse timestamp %q: %w", tt.TimeStamp, err)
		}
		transfers = append(transfers, transfer{
			Hash:  tt.Hash,
			From:  tt.From,
			To:    tt.To,
			Value: v,
			Time:  time.Unix(ts, 0).UTC(),
		})
	}
	return transfers, nil
}
