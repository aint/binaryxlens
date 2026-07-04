package internal

import (
	"fmt"
	"maps"
	"math/big"
	"slices"
	"strings"
)

const zeroAddr0x = "0x0000000000000000000000000000000000000000"

type Holder struct {
	Address string
	Balance *big.Int
}

type ProjectHolder struct {
	Address       string
	TokenBalances map[string]*big.Int
	TotalBalance  *big.Int
}

type Holders []Holder

func (t Token) getHolders() (Holders, error) {
	holderMap := make(map[string]*big.Int)
	for _, tx := range t.Txs {
		v, ok := new(big.Int).SetString(tx.Value, 10)
		if !ok {
			return nil, fmt.Errorf("parse value %q", tx.Value)
		}
		if tx.From != zeroAddr0x {
			cur := holderMap[tx.From]
			if cur == nil {
				cur = big.NewInt(0)
			}
			holderMap[tx.From] = new(big.Int).Sub(cur, v)
		}
		if tx.To != zeroAddr0x {
			cur := holderMap[tx.To]
			if cur == nil {
				cur = big.NewInt(0)
			}
			holderMap[tx.To] = new(big.Int).Add(cur, v)
		}
	}

	keys := slices.Collect(maps.Keys(holderMap))
	slices.SortFunc(keys, func(a, b string) int {
		return holderMap[b].Cmp(holderMap[a]) // descending by balance
	})

	holders := make(Holders, 0, len(holderMap))
	for _, address := range keys {
		if address == t.Address {
			// ignore token's balance
			continue
		}
		holders = append(holders, Holder{Address: address, Balance: holderMap[address]})
	}
	return holders, nil
}

func (t Token) PrintHolders(top int) error {
	idx := min(len(t.Holders), top)

	fmt.Printf("\nHolders: showing %d of %d\n", len(t.Holders[:idx]), len(t.Holders))
	fmt.Printf("%4s %-44s %32s %14s\n", "#", "address", "balance", "% of supply")
	for i, h := range t.Holders[:idx] {
		if h.Balance.Sign() == 0 {
			continue
		}
		fmt.Printf("%d. %s %32s %13s%%\n", i+1, h.Address, FormatBigInt(h.Balance, t.Decimal), PercentOf(h.Balance, t.TotalSupplyRaw))
	}

	return nil
}

func (p Project) getHolders() []ProjectHolder {
	projectHolderMap := make(map[string]*ProjectHolder)

	for _, token := range p.Tokens {
		for _, th := range token.Holders {
			ph := projectHolderMap[th.Address]
			if ph == nil {
				ph = &ProjectHolder{
					Address: th.Address,
					TokenBalances: map[string]*big.Int{token.Name: new(big.Int).Set(th.Balance)},
					TotalBalance: new(big.Int).Set(th.Balance),
				}
				projectHolderMap[th.Address] = ph
				continue
			}

			ph.TokenBalances[token.Name] = new(big.Int).Set(th.Balance)
			ph.TotalBalance.Add(ph.TotalBalance, th.Balance)
		}
	}

	projectHolders := make([]ProjectHolder, 0, len(projectHolderMap))
	for _, ph := range projectHolderMap {
		projectHolders = append(projectHolders, *ph)
	}
	slices.SortFunc(projectHolders, func(a, b ProjectHolder) int {
		return b.TotalBalance.Cmp(a.TotalBalance)
	})
	return projectHolders
}

func (p Project) PrintHolders(top int) error {
	idx := min(len(p.Holders), top)

	fmt.Printf("\nHolders: showing %d of %d\n", len(p.Holders[:idx]), len(p.Holders))
	fmt.Printf("%4s %-42s %-96s %12s %12s\n", "#", "address", "token names", "balance", "% supply")
	for i, h := range p.Holders[:idx] {
		names := slices.Sorted(maps.Keys(h.TokenBalances))
		fmt.Printf(
			"%4d %-42s %-96s %12s %11s%%\n",
			i+1,
			h.Address,
			strings.Join(names, ", "),
			FormatBigInt(h.TotalBalance, p.Decimal),
			PercentOf(h.TotalBalance, p.TotalSupplyRaw),
		)
	}

	return nil
}
