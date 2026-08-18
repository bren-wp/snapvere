package i18n

import (
	_ "embed"
	"encoding/json"
	"strings"
	"sync"
)

//go:embed messages.json
var messagesJSON []byte

type catalog struct {
	Languages map[string]map[string]string `json:"languages"`
}

var (
	loadOnce sync.Once
	loaded   catalog
)

func load() {
	loaded.Languages = map[string]map[string]string{}
	_ = json.Unmarshal(messagesJSON, &loaded)
}

// Resolve selects the best available language for a Windows locale such as
// "hr-HR" or "en-US". It always returns a usable translation table.
func Resolve(locale string) (string, map[string]string) {
	loadOnce.Do(load)
	code := "en"
	locale = strings.ReplaceAll(strings.TrimSpace(locale), "_", "-")
	if locale != "" {
		for k := range loaded.Languages {
			if strings.EqualFold(k, locale) {
				code = k
				break
			}
		}
		if code == "en" {
			base := locale
			if i := strings.Index(base, "-"); i >= 0 {
				base = base[:i]
			}
			for k := range loaded.Languages {
				if strings.EqualFold(k, base) {
					code = k
					break
				}
			}
		}
	}
	tr := loaded.Languages[code]
	if tr == nil {
		code = "en"
		tr = loaded.Languages[code]
	}
	if tr == nil {
		tr = map[string]string{}
	}
	return code, tr
}
