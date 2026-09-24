package locales

import (
	"embed"

	"github.com/BurntSushi/toml"
	"github.com/jeandeaual/go-locale"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

var Localizer *i18n.Localizer

var (
	//go:embed active.*.toml
	localeFS embed.FS
	bundle   *i18n.Bundle
)

func init() {
	locales, _ := locale.GetLocales()
	bundle = i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	bundle.LoadMessageFileFS(localeFS, "active.zh.toml")
	Localizer = i18n.NewLocalizer(bundle, locales...)
}
