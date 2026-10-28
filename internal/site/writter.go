package site

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"time"

)

//go:embed i18n.js
var i18nScript []byte

var (
	i18nPlaceholder         = []byte("__I18N_SCRIPT__")
	updatedStampPlaceholder = []byte("__UPDATED__")
)

func write(path string, updated time.Time, template, placeholder, payload []byte) error {
	if !bytes.Contains(template, placeholder) ||
		!bytes.Contains(template, updatedStampPlaceholder) ||
		!bytes.Contains(template, i18nPlaceholder) {
		return fmt.Errorf("missing placeholders")
	}

	out := bytes.ReplaceAll(withUpdatedStamp(withI18n(template), updated),
		placeholder,
		payload,
	)
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

// withI18n injects the shared dictionaries and t() helper into a page template.
func withI18n(tpl []byte) []byte {
	return bytes.ReplaceAll(tpl, i18nPlaceholder, i18nScript)
}

// withUpdatedStamp injects the updated at stamp
func withUpdatedStamp(tpl []byte, updated time.Time) []byte {
	stamp := updated.UTC().Format(reportsUpdatedLayout)
	return bytes.ReplaceAll(tpl, updatedStampPlaceholder, []byte(stamp))
}
