package theme

var (
	ColorNone     = "\033[0m"
	ColorBold     = "\033[1m"
	ColorGray     = "\033[0;90m"
	ColorArch     = "\033[0;94m"
	ColorStable   = "\033[0;92m"
	ColorTesting  = "\033[93m"
	ColorUnstable = "\033[0;91m"

	theme = map[rune]string{
		'a': ColorArch,
		's': ColorStable,
		't': ColorTesting,
		'u': ColorUnstable,
	}
)

func Theme(branch string) string {
	if len(branch) == 0 {
		return ColorNone

	}
	if _, ok := theme[rune(branch[0])]; !ok {
		return ColorNone
	}
	return theme[rune(branch[0])]
}
