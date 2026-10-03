package internal

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/aint/binaryxlens/internal/polygonscan"
)

type P2PTrade struct {
	Hash  string
	Buyer string
	Token string
	USDT  *big.Int // fee included
}

type p2pContract struct {
	Address    string
	TradeTopic string
	parseTrade func(polygonscan.EventLog) (P2PTrade, bool)
}

var p2pContracts = []p2pContract{
	{
		// used until 2026-08
		Address:    "0x4e209d2a48262ddde5525b9f0b2cdacabb156e97",
		TradeTopic: "0xb8979f8966efd325fe87225de92d041741755a633f6727c5d36045716572bfb4",
		parseTrade: parseLegacyP2PTrade,
	},
	{
		// used from 2026-08
		Address:    "0x706150702ac2f53c8321278fcdf659d96284d790",
		TradeTopic: "0xafa6b1f27e38bdefa56aee331174898e11b4d3cd76bf25ddb59a446d88ac7872",
		parseTrade: parseP2PFill,
	},
}

// FetchP2PTrades returns P2P trades of all tokens, keyed by token address.
func FetchP2PTrades(client *polygonscan.Client, scanPause time.Duration) (map[string][]P2PTrade, error) {
	tradesByToken := make(map[string][]P2PTrade)
	for _, c := range p2pContracts {
		logs, err := client.FetchLogs(c.Address, c.TradeTopic, scanPause)
		if err != nil {
			return nil, fmt.Errorf("fetch %s trades: %w", c.Address, err)
		}
		for _, l := range logs {
			if t, ok := c.parseTrade(l); ok {
				tradesByToken[t.Token] = append(tradesByToken[t.Token], t)
			}
		}
	}
	return tradesByToken, nil
}

// parseLegacyP2PTrade decodes the old contract's trade event. Its source is not
// published; the layout is inferred from receipts: data word 2 is the token,
// 12 the USDT paid incl. fee, 16 the buyer.
func parseLegacyP2PTrade(l polygonscan.EventLog) (P2PTrade, bool) {
	words := extractWords(l.Data)
	if len(words) < 17 {
		return P2PTrade{}, false
	}
	return P2PTrade{
		Hash:  l.TransactionHash,
		Buyer: wordAddress(words[16]),
		Token: wordAddress(words[2]),
		USDT:  wordInt(words[12]),
	}, true
}

// parseP2PFill decodes the new contract's fill event, emitted once per order side.
// Data words are tokenGive, tokenGet, amountGive, amountGet, fee; topics[2] is the
// order's account. Only the side giving USDT is a purchase.
func parseP2PFill(l polygonscan.EventLog) (P2PTrade, bool) {
	words := extractWords(l.Data)
	if len(words) < 3 || len(l.Topics) < 3 || wordAddress(words[0]) != polygonscan.USDTAddress {
		return P2PTrade{}, false
	}
	return P2PTrade{
		Hash:  l.TransactionHash,
		Buyer: wordAddress(strings.TrimPrefix(l.Topics[2], "0x")),
		Token: wordAddress(words[1]),
		USDT:  wordInt(words[2]),
	}, true
}

// extractWords splits ABI-encoded event data into 32-byte hex words.
func extractWords(data string) []string {
	data = strings.TrimPrefix(data, "0x")
	words := make([]string, 0, len(data)/64)
	for i := 0; i+64 <= len(data); i += 64 {
		words = append(words, data[i:i+64])
	}
	return words
}

func wordAddress(word string) string {
	return "0x" + strings.ToLower(word[24:])
}

func wordInt(word string) *big.Int {
	v, _ := new(big.Int).SetString(word, 16)
	return v
}
