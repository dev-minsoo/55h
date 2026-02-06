package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type AppTheme struct {
	Name          string
	TView         tview.Theme
	Bg            tcell.Color
	PanelBg       tcell.Color
	Border        tcell.Color
	Text          tcell.Color
	Muted         tcell.Color
	Accent        tcell.Color
	Success       tcell.Color
	Warning       tcell.Color
	Error         tcell.Color
	SelectBg      tcell.Color
	SelectFg      tcell.Color
	HeaderBg      tcell.Color
	FooterBg      tcell.Color
	Label         tcell.Color
	KeyChipFg     tcell.Color
	MarkupAccent  string
	MarkupSuccess string
	MarkupWarning string
	MarkupError   string
}

func DefaultThemes() []AppTheme {
	return []AppTheme{
		newTheme(
			"Dark",
			"#1a1b26",
			"#1f2335",
			"#3b4261",
			"#c0caf5",
			"#9aa5ce",
			"#7aa2f7",
			"#9ece6a",
			"#e0af68",
			"#f7768e",
			"#283457",
			"#ffffff",
			"#1a1b26",
			"#1a1b26",
			"#7dcfff",
			"#c0caf5",
		),
		newTheme(
			"Light",
			"#f7f7f7",
			"#ffffff",
			"#d0d7de",
			"#24292e",
			"#6e7781",
			"#0969da",
			"#1a7f37",
			"#9a6700",
			"#cf222e",
			"#ddf4ff",
			"#24292e",
			"#f7f7f7",
			"#f7f7f7",
			"#0550ae",
			"#24292e",
		),
		newTheme(
			"Neutral",
			"#2b2f37",
			"#313640",
			"#434a56",
			"#d7dae0",
			"#a0a6b2",
			"#8ab4f8",
			"#7ee787",
			"#f2cc60",
			"#ff7b72",
			"#3b4252",
			"#ffffff",
			"#2b2f37",
			"#2b2f37",
			"#9ecbff",
			"#d7dae0",
		),
		newTheme(
			"Solarized Dark",
			"#002b36",
			"#073642",
			"#586e75",
			"#839496",
			"#93a1a1",
			"#268bd2",
			"#859900",
			"#b58900",
			"#dc322f",
			"#073642",
			"#fdf6e3",
			"#002b36",
			"#002b36",
			"#268bd2",
			"#93a1a1",
		),
		newTheme(
			"Solarized Light",
			"#fdf6e3",
			"#eee8d5",
			"#93a1a1",
			"#657b83",
			"#839496",
			"#268bd2",
			"#859900",
			"#b58900",
			"#dc322f",
			"#eee8d5",
			"#002b36",
			"#fdf6e3",
			"#fdf6e3",
			"#268bd2",
			"#657b83",
		),
		newTheme(
			"Gruvbox Dark",
			"#282828",
			"#3c3836",
			"#504945",
			"#ebdbb2",
			"#a89984",
			"#fb4934",
			"#b8bb26",
			"#fabd2f",
			"#cc241d",
			"#3c3836",
			"#fbf1c7",
			"#282828",
			"#282828",
			"#fb4934",
			"#ebdbb2",
		),
		newTheme(
			"Gruvbox Light",
			"#fbf1c7",
			"#f2e5bc",
			"#d5c4a1",
			"#3c3836",
			"#7c6f64",
			"#b57614",
			"#98971a",
			"#b57614",
			"#cc241d",
			"#f2e5bc",
			"#3c3836",
			"#fbf1c7",
			"#fbf1c7",
			"#b57614",
			"#3c3836",
		),
		newTheme(
			"Dracula",
			"#282a36",
			"#44475a",
			"#6272a4",
			"#f8f8f2",
			"#6272a4",
			"#bd93f9",
			"#50fa7b",
			"#f1fa8c",
			"#ff79c6",
			"#44475a",
			"#f8f8f2",
			"#282a36",
			"#282a36",
			"#bd93f9",
			"#f8f8f2",
		),
		newTheme(
			"Nord",
			"#2e3440",
			"#3b4252",
			"#4c566a",
			"#d8dee9",
			"#88c0d0",
			"#81a1c1",
			"#a3be8c",
			"#ebcb8b",
			"#bf616a",
			"#3b4252",
			"#eceff4",
			"#2e3440",
			"#2e3440",
			"#81a1c1",
			"#d8dee9",
		),
		newTheme(
			"Oceanic",
			"#1b2b34",
			"#223b44",
			"#2b6f77",
			"#d3e0ea",
			"#7aa2c6",
			"#2ac3de",
			"#9ad66a",
			"#f6c177",
			"#ff6b6b",
			"#223b44",
			"#eaf6ff",
			"#1b2b34",
			"#1b2b34",
			"#2ac3de",
			"#d3e0ea",
		),
		newTheme(
			"Monokai",
			"#272822",
			"#3e3d32",
			"#5f5a60",
			"#f8f8f2",
			"#a39e9a",
			"#fd971f",
			"#a6e22e",
			"#e6db74",
			"#f92672",
			"#3e3d32",
			"#272822",
			"#272822",
			"#272822",
			"#fd971f",
			"#f8f8f2",
		),
		newTheme(
			"HighContrast",
			"#000000",
			"#000000",
			"#ffffff",
			"#ffffff",
			"#bfbfbf",
			"#00ff00",
			"#00ffff",
			"#ffff00",
			"#ff0000",
			"#ffffff",
			"#000000",
			"#000000",
			"#000000",
			"#00ff00",
			"#00ff00",
		),
	}
}

func newTheme(
	name string,
	bg string,
	panel string,
	border string,
	text string,
	muted string,
	accent string,
	success string,
	warning string,
	error string,
	selectBg string,
	selectFg string,
	headerBg string,
	footerBg string,
	label string,
	keyChipFg string,
) AppTheme {
	bgColor := hexColor(bg)
	panelColor := hexColor(panel)
	borderColor := hexColor(border)
	textColor := hexColor(text)
	mutedColor := hexColor(muted)
	accentColor := hexColor(accent)
	successColor := hexColor(success)
	warningColor := hexColor(warning)
	errorColor := hexColor(error)
	selectBgColor := hexColor(selectBg)
	selectFgColor := hexColor(selectFg)
	headerBgColor := hexColor(headerBg)
	footerBgColor := hexColor(footerBg)
	labelColor := hexColor(label)
	keyChipFgColor := hexColor(keyChipFg)

	return AppTheme{
		Name:      name,
		Bg:        bgColor,
		PanelBg:   panelColor,
		Border:    borderColor,
		Text:      textColor,
		Muted:     mutedColor,
		Accent:    accentColor,
		Success:   successColor,
		Warning:   warningColor,
		Error:     errorColor,
		SelectBg:  selectBgColor,
		SelectFg:  selectFgColor,
		HeaderBg:  headerBgColor,
		FooterBg:  footerBgColor,
		Label:     labelColor,
		KeyChipFg: keyChipFgColor,
		TView: tview.Theme{
			PrimitiveBackgroundColor:    bgColor,
			ContrastBackgroundColor:     panelColor,
			MoreContrastBackgroundColor: selectBgColor,
			BorderColor:                 borderColor,
			TitleColor:                  accentColor,
			GraphicsColor:               accentColor,
			PrimaryTextColor:            textColor,
			SecondaryTextColor:          mutedColor,
			TertiaryTextColor:           successColor,
			InverseTextColor:            selectFgColor,
			ContrastSecondaryTextColor:  warningColor,
		},
		MarkupAccent:  accent,
		MarkupSuccess: success,
		MarkupWarning: warning,
		MarkupError:   error,
	}
}

func hexColor(value string) tcell.Color {
	clean := strings.TrimPrefix(value, "#")
	if len(clean) != 6 {
		return tcell.ColorDefault
	}

	parsed, err := strconv.ParseInt(clean, 16, 32)
	if err != nil {
		return tcell.ColorDefault
	}

	r := int32((parsed >> 16) & 0xff)
	g := int32((parsed >> 8) & 0xff)
	b := int32(parsed & 0xff)
	return tcell.NewRGBColor(r, g, b)
}

func ASCIILogo() string {
	return strings.TrimRight(fmt.Sprintf(`[#7aa2f7]  ______  ______  __   _[-:-:-]  
[#9ece6a] |  ____||  ____||  |_| |[-:-:-] 
[#e0af68] |___   \|___   \|   _  |[-:-:-] 
[#f7768e] |______/|______/|__| |_|[-:-:-]
`), "\n")
}
