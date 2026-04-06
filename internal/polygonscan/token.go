package polygonscan

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"strconv"
	"strings"
)

type TokenMetadata struct {
	TotalSupply *big.Int
	Decimals    uint8
}

func (c *Client) GetTokenMetadata(addr string) (TokenMetadata, error) {
	supply, err := c.tokenSupply(addr)
	if err != nil {
		return TokenMetadata{}, fmt.Errorf("token supply: %w", err)
	}
	decimals, err := c.decimalsFromFirstTokenTx(addr)
	if err != nil {
		return TokenMetadata{}, fmt.Errorf("decimals: %w", err)
	}

	return TokenMetadata{
		TotalSupply: supply,
		Decimals:    decimals,
	}, nil
}

func (c *Client) tokenSupply(addr string) (*big.Int, error) {
	q := url.Values{}
	q.Set("module", "stats")
	q.Set("action", "tokensupply")
	q.Set("contractaddress", addr)

	raw, err := c.get(q)
	if err != nil {
		return nil, err
	}
	var resp response
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if resp.Status != "1" {
		return nil, fmt.Errorf("api status=%s, message=%s, detail=%s", resp.Status, resp.Message, resp.Result)
	}
	n := big.NewInt(0)
	if _, ok := n.SetString(resp.Result, 10); !ok {
		return nil, fmt.Errorf("parse supply %s", resp.Result)
	}
	return n, nil
}

func (c *Client) decimalsFromFirstTokenTx(addr string) (uint8, error) {
	rows, err := c.tokenTxPage(addr, 1, 1, "asc")
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, errors.New("no tokentx to infer decimals")
	}
	s := strings.TrimSpace(rows[0].TokenDecimal)
	if s == "" {
		return 0, fmt.Errorf("tokenDecimal missing on tokentx row")
	}
	n, err := strconv.ParseUint(s, 10, 8)
	if err != nil {
		return 0, fmt.Errorf("parse tokenDecimal %q: %w", s, err)
	}
	return uint8(n), nil
}

type response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  string `json:"result"`
}
