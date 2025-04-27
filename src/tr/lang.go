package tr

import (
	"embed"
	"os"

	"github.com/leonelquinteros/gotext"
)

var (
	//go:embed LC_MESSAGES/*.po
	poFs embed.FS

	Lg Lang
)

type Lang struct {
	lang string
	Po   *gotext.Po
}

func (l Lang) T(str string, vars ...interface{}) string {
	return l.Po.Get(str, vars...)
}

func NewLang() Lang {
	lang := os.Getenv("LANG")
	if lc := os.Getenv("LANGUAGE"); lc != "" {
		lang = lc
	} else if lc := os.Getenv("LC_ALL"); lc != "" {
		lang = lc
	}
	if len(lang) > 1 {
		lang = lang[:2]
	} else {
		lang = "en"
	}

	ret := Lang{
		lang: lang,
		Po:   gotext.NewPoFS(poFs),
	}
	ret.Po.ParseFile("LC_MESSAGES/" + lang + ".po")

	return ret
}

func T(str string, vars ...interface{}) string {
	return Lg.T(str, vars...)
}

func init() {
	Lg = NewLang()
}
