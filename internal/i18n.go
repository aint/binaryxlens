package internal

import (
	"bytes"
	_ "embed"
)

//go:embed i18n.js
var i18nScript []byte

var i18nPlaceholder = []byte("__I18N_SCRIPT__")

// withI18n injects the shared dictionaries and t() helper into a page template.
func withI18n(tpl []byte) []byte {
	return bytes.ReplaceAll(tpl, i18nPlaceholder, i18nScript)
}
