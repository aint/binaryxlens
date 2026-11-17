package site

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"strings"
	"time"
)

//go:embed i18n.js
var i18nScript []byte

//go:embed header.html
var commonHeaderTemplate []byte

var (
	i18nPlaceholder             = []byte("__I18N_SCRIPT__")
	updatedStampPlaceholder     = []byte("__UPDATED__")
	commonHeaderPlaceholder     = []byte("__COMMON_HEADER__")
	rootPrefixPlaceholder       = []byte("__ROOT_PREFIX__")
	projectsSelectedPlaceholder = []byte("__PROJECTS_SELECTED__")
	holdersSelectedPlaceholder  = []byte("__HOLDERS_SELECTED__")
	trendsSelectedPlaceholder   = []byte("__TRENDS_SELECTED__")
)

type headerConfig struct {
	activePage string
	rootPrefix string
}

func write(path string, updated time.Time, template, placeholder, payload []byte, header headerConfig) error {
	if !bytes.Contains(template, placeholder) ||
		!bytes.Contains(template, commonHeaderPlaceholder) ||
		!bytes.Contains(template, i18nPlaceholder) {
		return fmt.Errorf("missing placeholders")
	}

	out := []byte(strings.NewReplacer(
		string(commonHeaderPlaceholder), string(withCommonHeader(header, updated)),
		string(i18nPlaceholder), string(i18nScript),
		string(placeholder), string(payload),
	).Replace(string(template)))
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

// withCommonHeader injects the common header and update stamp.
func withCommonHeader(config headerConfig, updated time.Time) []byte {
	header := bytes.ReplaceAll(commonHeaderTemplate, rootPrefixPlaceholder, []byte(config.rootPrefix))
	selected := map[string][]byte{
		"projects": projectsSelectedPlaceholder,
		"holders":  holdersSelectedPlaceholder,
		"trends":   trendsSelectedPlaceholder,
	}
	for page, placeholder := range selected {
		value := []byte("false")
		if config.activePage == page {
			value = []byte("true")
		}
		header = bytes.ReplaceAll(header, placeholder, value)
	}
	stamp := updated.UTC().Format(reportsUpdatedLayout)
	header = bytes.ReplaceAll(header, updatedStampPlaceholder, []byte(stamp))
	return header
}

// withI18n injects the shared dictionaries and t() helper into a page template.
func withI18n(tpl []byte) []byte {
	return bytes.ReplaceAll(tpl, i18nPlaceholder, i18nScript)
}
