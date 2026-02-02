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

	return AppTheme{
		Name:     name,
		Bg:       bgColor,
		PanelBg:  panelColor,
		Border:   borderColor,
		Text:     textColor,
		Muted:    mutedColor,
		Accent:   accentColor,
		Success:  successColor,
		Warning:  warningColor,
		Error:    errorColor,
		SelectBg: selectBgColor,
		SelectFg: selectFgColor,
		HeaderBg: headerBgColor,
		FooterBg: footerBgColor,
		Label:    labelColor,
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
