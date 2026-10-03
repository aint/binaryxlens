package polygonscan

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

const logsPageSize = 1000

// EventLog is one row from module=logs&action=getLogs.
type EventLog struct {
	Topics          []string `json:"topics"`
	Data            string   `json:"data"`
	BlockNumber     string   `json:"blockNumber"`
	LogIndex        string   `json:"logIndex"`
	TransactionHash string   `json:"transactionHash"`
}

// FetchLogs returns every event with topic0 emitted by address.
// getLogs pages cap at 1000 rows, so each request starts at the last block seen
// and rows repeated from that block are dropped.
func (c *Client) FetchLogs(address, topic0 string, pause time.Duration) ([]EventLog, error) {
	var all []EventLog
	seen := make(map[string]bool)
	fromBlock := int64(0)
	for {
		batch, err := c.logsPage(address, topic0, fromBlock)
		if err != nil {
			return all, err
		}
		for _, l := range batch {
			id := l.TransactionHash + l.LogIndex
			if !seen[id] {
				seen[id] = true
				all = append(all, l)
			}
		}
		if len(batch) < logsPageSize {
			return all, nil
		}
		lastBlock, err := strconv.ParseInt(batch[len(batch)-1].BlockNumber, 0, 64)
		if err != nil {
			return all, fmt.Errorf("parse block number %q: %w", batch[len(batch)-1].BlockNumber, err)
		}
		if lastBlock == fromBlock {
			return all, fmt.Errorf("block %d has more than %d logs", fromBlock, logsPageSize)
		}
		fromBlock = lastBlock
		if pause > 0 {
			time.Sleep(pause)
		}
	}
}

func (c *Client) logsPage(address, topic0 string, fromBlock int64) ([]EventLog, error) {
	q := url.Values{}
	q.Set("module", "logs")
	q.Set("action", "getLogs")
	q.Set("address", address)
	q.Set("topic0", topic0)
	q.Set("fromBlock", strconv.FormatInt(fromBlock, 10))
	q.Set("toBlock", "latest")
	q.Set("page", "1")
	q.Set("offset", strconv.Itoa(logsPageSize))

	body, err := c.get(q)
	if err != nil {
		return nil, err
	}

	var envelope struct {
		Status  string          `json:"status"`
		Message string          `json:"message"`
		Result  json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("decode envelope: %w", err)
	}
	if envelope.Status != "1" {
		if envelope.Message == "No records found" {
			return nil, nil
		}
		return nil, fmt.Errorf("polygonscan api: status=%s message=%s result=%s", envelope.Status, envelope.Message, envelope.Result)
	}

	var logs []EventLog
	if err := json.Unmarshal(envelope.Result, &logs); err != nil {
		return nil, fmt.Errorf("decode getLogs rows: %w", err)
	}
	return logs, nil
}
