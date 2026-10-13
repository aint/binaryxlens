package internal

import (
	"bytes"
	_ "embed"
	"fmt"
	"html"
	"math/big"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/aint/binaryxlens/internal/polygonscan"
)

//go:embed signals.html
var signalsTemplateSrc []byte

var signalsPlaceholder = []byte("__SIGNALS__")

// SignalKind is one feed event.
type SignalKind string

const (
	SignalInsiderIn      SignalKind = "insider_in"
	SignalInsiderOut     SignalKind = "insider_out"
	SignalNewHolder      SignalKind = "new_holder"
	SignalExitedHolder   SignalKind = "exited_holder"
	SignalNewInvestor    SignalKind = "new_investor"
	SignalExitedInvestor SignalKind = "exited_investor"
)

// Signal is one watched-wallet move or a balance crossing zero.
type Signal struct {
	Time     time.Time
	Kind     SignalKind
	Wallet   string
	Label    string
	Project  string
	Property string
	Tokens   *big.Int
	USDT     *big.Int
	Hash     string
}

// watchedWallets are platform staff, keyed by lowercase address.
var watchedWallets = map[string]string{
	"0xdf24253c84f16aac030917c2672720b11610e109": "Oleh",
}

type signalLeg struct {
	project  string
	property *Property
	transfer transfer
}

// CollectSignals replays every transfer so balances stay correct, then returns
// events at or after since, newest first.
func CollectSignals(projects []*Project, since time.Time) []Signal {
	propertyBalance := make(map[string]*big.Int)
	totalBalance := make(map[string]*big.Int)
	var signals []Signal

	for _, leg := range signalLegs(projects) {
		t := leg.transfer
		if t.Value == nil || t.Value.Sign() == 0 || t.From == t.To {
			continue
		}
		if holderWallet(t.From, leg.property) {
			delta := new(big.Int).Neg(t.Value)
			signals = append(signals, legSignals(leg, t.From, delta, since, propertyBalance, totalBalance)...)
		}
		if holderWallet(t.To, leg.property) {
			signals = append(signals, legSignals(leg, t.To, t.Value, since, propertyBalance, totalBalance)...)
		}
	}

	slices.SortFunc(signals, func(a, b Signal) int {
		if c := b.Time.Compare(a.Time); c != 0 {
			return c
		}
		if c := strings.Compare(a.Hash, b.Hash); c != 0 {
			return c
		}
		return strings.Compare(string(a.Kind), string(b.Kind))
	})
	return signals
}

func signalLegs(projects []*Project) []signalLeg {
	var legs []signalLeg
	for _, project := range projects {
		for _, property := range project.Properties {
			for _, t := range property.transfers {
				legs = append(legs, signalLeg{project: project.Name, property: property, transfer: t})
			}
		}
	}
	slices.SortFunc(legs, func(a, b signalLeg) int {
		if c := a.transfer.Time.Compare(b.transfer.Time); c != 0 {
			return c
		}
		return strings.Compare(a.transfer.Hash, b.transfer.Hash)
	})
	return legs
}

func holderWallet(addr string, property *Property) bool {
	return addr != zeroAddr0x && addr != property.Address
}

func legSignals(leg signalLeg, wallet string, delta *big.Int, since time.Time, propertyBalance, totalBalance map[string]*big.Int) []Signal {
	propKey := wallet + "\x00" + leg.property.Address
	beforeProp, afterProp := addBalance(propertyBalance, propKey, delta)
	beforeTotal, afterTotal := addBalance(totalBalance, wallet, delta)
	if leg.transfer.Time.Before(since) {
		return nil
	}

	t := leg.transfer
	base := Signal{
		Time:     t.Time,
		Wallet:   wallet,
		Label:    watchedWallets[wallet],
		Project:  leg.project,
		Property: leg.property.Name,
		Tokens:   new(big.Int).Set(t.Value),
		Hash:     t.Hash,
	}
	if t.USDT != nil {
		base.USDT = new(big.Int).Set(t.USDT)
	}

	var out []Signal
	if _, watched := watchedWallets[wallet]; watched {
		kind := SignalInsiderIn
		if delta.Sign() < 0 {
			kind = SignalInsiderOut
		}
		out = append(out, withKind(base, kind))
	}
	if !positive(beforeProp) && positive(afterProp) {
		out = append(out, withKind(base, SignalNewHolder))
	}
	if positive(beforeProp) && !positive(afterProp) {
		out = append(out, withKind(base, SignalExitedHolder))
	}
	if !positive(beforeTotal) && positive(afterTotal) {
		out = append(out, withKind(base, SignalNewInvestor))
	}
	if positive(beforeTotal) && !positive(afterTotal) {
		out = append(out, withKind(base, SignalExitedInvestor))
	}
	return out
}

func withKind(s Signal, kind SignalKind) Signal {
	s.Kind = kind
	return s
}

func positive(n *big.Int) bool {
	return n.Sign() > 0
}

// WriteSignals writes the feed for events at or after since, newest first.
func WriteSignals(path string, projects []*Project, since time.Time) error {
	if !bytes.Contains(signalsTemplateSrc, signalsPlaceholder) || !bytes.Contains(signalsTemplateSrc, i18nPlaceholder) {
		return fmt.Errorf("signals template missing placeholder")
	}
	feed := signalsFeedHTML(CollectSignals(projects, since), propertyDecimals(projects))
	out := bytes.ReplaceAll(withI18n(signalsTemplateSrc), signalsPlaceholder, []byte(feed))
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return fmt.Errorf("write signals: %w", err)
	}
	fmt.Println("Signals are ready at", path)
	return nil
}

func propertyDecimals(projects []*Project) map[string]uint8 {
	out := make(map[string]uint8)
	for _, project := range projects {
		for _, property := range project.Properties {
			out[project.Name+"\x00"+property.Name] = property.Decimal
		}
	}
	return out
}

func signalsFeedHTML(signals []Signal, decimals map[string]uint8) string {
	var b strings.Builder
	for _, s := range signals {
		dir := "in"
		if !signalInbound(s.Kind) {
			dir = "out"
		}
		name := s.Label
		if name == "" {
			name = shortWallet(s.Wallet)
		}
		addr := ""
		if s.Label != "" {
			addr = fmt.Sprintf(` <span class="addr">%s</span>`, html.EscapeString(shortWallet(s.Wallet)))
		}
		usdt := ""
		if s.USDT != nil {
			usdt = fmt.Sprintf(`<span class="usdt"> · %s USDT</span>`, html.EscapeString(FormatBigInt(s.USDT, polygonscan.USDTDecimal)))
		}
		where := s.Project
		if s.Property != "" && s.Property != s.Project {
			where = s.Project + " · " + s.Property
		}
		text := strings.ToLower(strings.TrimSpace(strings.Join([]string{s.Label, s.Wallet, s.Project, s.Property, string(s.Kind)}, " ")))
		fmt.Fprintf(&b, `
			<article class="signal %s" data-kind="%s" data-text="%s">
				<div class="meta">
					<time datetime="%s">%s UTC</time>
					<span class="tag tag-%s" data-i18n="signal.%s">%s</span>
				</div>
				<p class="who"><a href="https://polygonscan.com/address/%s" title="%s">%s</a>%s</p>
				<p class="where">%s</p>
				<p class="amt"><span class="num">%s</span> <span data-i18n="signal.tokens">tokens</span>%s</p>
				<p class="tx"><a href="https://polygonscan.com/tx/%s" data-i18n="signal.tx">tx</a></p>
			</article>`,
			dir,
			html.EscapeString(string(s.Kind)),
			html.EscapeString(text),
			html.EscapeString(s.Time.UTC().Format(time.RFC3339)),
			html.EscapeString(s.Time.UTC().Format(reportsUpdatedLayout)),
			html.EscapeString(string(s.Kind)),
			html.EscapeString(string(s.Kind)),
			html.EscapeString(string(s.Kind)),
			html.EscapeString(s.Wallet),
			html.EscapeString(s.Wallet),
			html.EscapeString(name),
			addr,
			html.EscapeString(where),
			html.EscapeString(FormatBigInt(s.Tokens, decimals[s.Project+"\x00"+s.Property])),
			usdt,
			html.EscapeString(s.Hash),
		)
	}
	return b.String()
}

func signalInbound(kind SignalKind) bool {
	switch kind {
	case SignalInsiderIn, SignalNewHolder, SignalNewInvestor:
		return true
	default:
		return false
	}
}

func shortWallet(addr string) string {
	if len(addr) < 12 {
		return addr
	}
	return addr[:6] + "…" + addr[len(addr)-4:]
}

func addBalance(m map[string]*big.Int, key string, delta *big.Int) (before, after *big.Int) {
	cur := m[key]
	if cur == nil {
		cur = new(big.Int)
		m[key] = cur
	}
	before = new(big.Int).Set(cur)
	cur.Add(cur, delta)
	return before, cur
}
