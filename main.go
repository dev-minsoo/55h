package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type HostEntry struct {
	Patterns            []string
	HostName            string
	User                string
	Port                string
	IdentityFile        string
	ProxyJump           string
	ServerAliveInterval *int
	ServerAliveCountMax *int
	ForwardAgent        *bool
	IdentitiesOnly      *bool
}

func (entry HostEntry) SearchText() string {
	parts := []string{
		strings.Join(entry.Patterns, " "),
		entry.HostName,
		entry.User,
		entry.Port,
		entry.IdentityFile,
		entry.ProxyJump,
	}
	if entry.ServerAliveInterval != nil {
		parts = append(parts, fmt.Sprintf("%d", *entry.ServerAliveInterval))
	}
	if entry.ServerAliveCountMax != nil {
		parts = append(parts, fmt.Sprintf("%d", *entry.ServerAliveCountMax))
	}
	if entry.ForwardAgent != nil {
		parts = append(parts, fmt.Sprintf("%t", *entry.ForwardAgent))
	}
	if entry.IdentitiesOnly != nil {
		parts = append(parts, fmt.Sprintf("%t", *entry.IdentitiesOnly))
	}
	return strings.ToLower(strings.Join(parts, " "))
}

func (entry HostEntry) DisplayText() (string, string) {
	mainText := "(unnamed)"
	if len(entry.Patterns) > 0 {
		mainText = entry.Patterns[0]
	}

	primary := mainText
	if entry.User != "" && entry.HostName != "" {
		primary = fmt.Sprintf("%s  %s@%s", primary, entry.User, entry.HostName)
	} else if entry.HostName != "" {
		primary = fmt.Sprintf("%s  %s", primary, entry.HostName)
	} else if entry.User != "" {
		primary = fmt.Sprintf("%s  %s", primary, entry.User)
	}

	if entry.Port != "" {
		primary = fmt.Sprintf("%s :%s", primary, entry.Port)
	}
	if entry.ProxyJump != "" {
		primary = fmt.Sprintf("%s via %s", primary, entry.ProxyJump)
	}

	return primary, ""
}

type AppConfig struct {
	ThemeName string `json:"theme_name"`
}

type AppState struct {
	App            *tview.Application
	Pages          *tview.Pages
	Root           *tview.Flex
	Header         *tview.Flex
	HeaderLogo     *tview.TextView
	HeaderMeta     *tview.TextView
	Footer         *tview.TextView
	SearchInput    *tview.InputField
	HostList       *tview.List
	DetailTable    *tview.Table
	ConfigPath     string
	Entries        []HostEntry
	Filtered       []HostEntry
	CurrentIndex   int
	ThemeIndex     int
	ThemeCatalog   []AppTheme
	CurrentFilter  string
	LastUpdated    time.Time
	LastLoadErr    error
	ThemeModalOpen bool
}

const (
	appVersion = "0.1.0"
	githubURL  = "https://github.com/"
)

func main() {
	configPath := resolveConfigPath()

	// CLI: support "55h add ssh ..." before launching TUI
	if len(os.Args) >= 3 && os.Args[1] == "add" && os.Args[2] == "ssh" {
		if err := handleAddSSH(os.Args[3:], configPath); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	app := tview.NewApplication()
	pages := tview.NewPages()
	root := tview.NewFlex().SetDirection(tview.FlexRow)
	content := tview.NewFlex().SetDirection(tview.FlexColumn)
	header := tview.NewFlex().SetDirection(tview.FlexColumn)
	headerLogo := tview.NewTextView()
	headerMeta := tview.NewTextView()
	footer := tview.NewTextView()
	searchInput := tview.NewInputField()
	hostList := tview.NewList()
	detailTable := tview.NewTable()

	state := &AppState{
		App:            app,
		Pages:          pages,
		Root:           root,
		Header:         header,
		HeaderLogo:     headerLogo,
		HeaderMeta:     headerMeta,
		Footer:         footer,
		SearchInput:    searchInput,
		HostList:       hostList,
		DetailTable:    detailTable,
		ConfigPath:     configPath,
		Entries:        nil,
		Filtered:       nil,
		CurrentIndex:   0,
		ThemeCatalog:   DefaultThemes(),
		ThemeIndex:     0,
		ThemeModalOpen: false,
	}

	// Load saved theme
	state.loadAppConfig()

	setupHeader(header, headerLogo, headerMeta)
	setupFooter(footer)
	setupSearchInput(searchInput, state)
	setupHostList(hostList)
	setupDetailTable(detailTable)

	leftPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	leftPanel.AddItem(searchInput, 3, 0, false)
	leftPanel.AddItem(hostList, 0, 1, true)

	content.AddItem(leftPanel, 0, 2, true)
	content.AddItem(detailTable, 0, 3, false)

	root.AddItem(header, 5, 0, false)
	root.AddItem(content, 0, 1, true)
	root.AddItem(footer, 3, 0, false)

	pages.AddPage("main", root, true, true)

	state.applyTheme(state.ThemeCatalog[state.ThemeIndex])
	state.reload()

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// If theme modal is open, don't process global shortcuts
		if state.ThemeModalOpen {
			return event
		}

		// Check if SearchInput has focus - only allow Escape to exit search mode
		searchFocused := state.App.GetFocus() == state.SearchInput

		switch event.Key() {
		case tcell.KeyCtrlC:
			app.Stop()
			return nil
		case tcell.KeyEsc:
			state.App.SetFocus(state.HostList)
			return nil
		case tcell.KeyEnter:
			if !searchFocused {
				state.connectSSH()
				return nil
			}
		}

		// Skip rune-based commands when search input is focused
		if searchFocused {
			return event
		}

		switch event.Rune() {
		case 't':
			state.showThemeModal()
			return nil
		case '?':
			state.showHelpModal()
			return nil
		case ':':
			state.App.SetFocus(state.SearchInput)
			return nil
		}

		return event
	})

	if err := app.SetRoot(pages, true).EnableMouse(true).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func resolveConfigPath() string {
	if custom := strings.TrimSpace(os.Getenv("SSH_CONFIG")); custom != "" {
		return custom
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, ".ssh", "config")
}

func setupHeader(header *tview.Flex, logo *tview.TextView, meta *tview.TextView) {
	logo.SetDynamicColors(true)
	logo.SetTextAlign(tview.AlignLeft)
	logo.SetWordWrap(false)
	logo.SetText(ASCIILogo())

	meta.SetDynamicColors(true)
	meta.SetTextAlign(tview.AlignRight)
	meta.SetWordWrap(false)

	header.SetBorder(false)
	header.AddItem(logo, 0, 2, false)
	header.AddItem(meta, 0, 3, false)
}

func setupFooter(footer *tview.TextView) {
	footer.SetDynamicColors(true)
	footer.SetTextAlign(tview.AlignCenter)
	footer.SetBorder(true)
}

func setupSearchInput(input *tview.InputField, state *AppState) {
	input.SetLabel("🔍 ")
	input.SetFieldWidth(0)
	input.SetPlaceholder("Filter hosts")
	input.SetBorder(true)
	input.SetTitle(" Search ")
	input.SetTitleAlign(tview.AlignLeft)
	input.SetDoneFunc(func(key tcell.Key) {
		state.App.SetFocus(state.HostList)
	})
	input.SetChangedFunc(func(text string) {
		state.CurrentFilter = text
		state.applyFilter(text)
	})
}

func setupHostList(hostList *tview.List) {
	hostList.ShowSecondaryText(false)
	hostList.SetBorder(true)
	hostList.SetTitle(" Hosts ")
	hostList.SetHighlightFullLine(true)
}

func setupDetailTable(detailTable *tview.Table) {
	detailTable.SetBorder(true)
	detailTable.SetTitle(" Details ")
	detailTable.SetSelectable(false, false)
}

func (state *AppState) reload() {
	entries, err := loadSSHConfig(state.ConfigPath)
	state.Entries = entries
	state.LastUpdated = time.Now()
	state.LastLoadErr = err
	state.applyFilter(state.CurrentFilter)
	state.updateHeaderMeta(state.LastUpdated, state.LastLoadErr)
}

func (state *AppState) applyFilter(query string) {
	state.Filtered = nil
	state.CurrentIndex = 0
	state.HostList.Clear()

	for _, entry := range state.Entries {
		if !fuzzyMatch(query, entry.SearchText()) {
			continue
		}
		state.Filtered = append(state.Filtered, entry)
		mainText, secondary := entry.DisplayText()
		state.HostList.AddItem(mainText, secondary, 0, nil)
	}

	state.HostList.SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		state.CurrentIndex = index
		state.renderDetails(index)
		state.updateHeaderMeta(state.LastUpdated, state.LastLoadErr)
	})
	state.HostList.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		state.CurrentIndex = index
		state.renderDetails(index)
		state.updateHeaderMeta(state.LastUpdated, state.LastLoadErr)
	})

	if len(state.Filtered) > 0 {
		state.renderDetails(0)
	} else {
		state.DetailTable.Clear()
		state.DetailTable.SetTitle(" Details ")
	}

	state.updateHeaderMeta(state.LastUpdated, state.LastLoadErr)
}

func (state *AppState) updateHeaderMeta(updatedAt time.Time, loadErr error) {
	themeName := state.ThemeCatalog[state.ThemeIndex].Name
	updatedAtText := updatedAt.Format("15:04:05")
	configShort := shortenPath(state.ConfigPath, 36)

	_ = themeName
	_ = updatedAtText
	_ = configShort

	state.HeaderMeta.SetText(fmt.Sprintf("[::b]SSH Manager[-:-:-]\n[::b][%s]v%s[-:-:-]\n[%s]%s[-:-:-]\n[%s]55h[-:-:-]",
		state.currentTheme().MarkupAccent,
		appVersion,
		state.currentTheme().MarkupAccent,
		githubURL,
		state.currentTheme().MarkupAccent,
	))
}

func (state *AppState) renderDetails(index int) {
	if index < 0 || index >= len(state.Filtered) {
		return
	}

	entry := state.Filtered[index]
	state.DetailTable.Clear()
	state.DetailTable.SetTitle(fmt.Sprintf(" Details: %s ", strings.Join(entry.Patterns, ", ")))

	rows := [][2]string{
		{"HostName", entry.HostName},
		{"User", entry.User},
		{"Port", entry.Port},
		{"IdentityFile", entry.IdentityFile},
		{"ProxyJump", entry.ProxyJump},
	}

	// Append optional fields when present
	if entry.ServerAliveInterval != nil {
		rows = append(rows, [2]string{"ServerAliveInterval", fmt.Sprintf("%d", *entry.ServerAliveInterval)})
	}
	if entry.ServerAliveCountMax != nil {
		rows = append(rows, [2]string{"ServerAliveCountMax", fmt.Sprintf("%d", *entry.ServerAliveCountMax)})
	}
	if entry.ForwardAgent != nil {
		rows = append(rows, [2]string{"ForwardAgent", fmt.Sprintf("%t", *entry.ForwardAgent)})
	}
	if entry.IdentitiesOnly != nil {
		rows = append(rows, [2]string{"IdentitiesOnly", fmt.Sprintf("%t", *entry.IdentitiesOnly)})
	}

	for i, row := range rows {
		labelCell := tview.NewTableCell("[::b]" + row[0])
		valueCell := tview.NewTableCell(row[1])
		labelCell.SetTextColor(state.currentTheme().Label)
		valueCell.SetTextColor(state.currentTheme().Text)
		valueCell.SetExpansion(1)
		state.DetailTable.SetCell(i, 0, labelCell)
		state.DetailTable.SetCell(i, 1, valueCell)
	}
}

func (state *AppState) showThemeModal() {
	state.ThemeModalOpen = true
	// Disable mouse to prevent background clicks while modal is open
	state.App.EnableMouse(false)
	originalThemeIndex := state.ThemeIndex

	modal := tview.NewList()
	// List itself won't draw a border; we'll wrap it in a bordered container
	modal.SetBorder(false)
	modal.ShowSecondaryText(false)
	modal.SetHighlightFullLine(true)

	// Style the modal with current theme
	updateModalStyle := func() {
		theme := state.currentTheme()
		modal.SetBackgroundColor(theme.PanelBg)
		modal.SetBorderColor(theme.Border)
		// Title and border will be applied on the surrounding box
		modal.SetMainTextStyle(tcell.StyleDefault.Foreground(theme.Text).Background(theme.PanelBg))
		// Use selection colors consistent with help modal chips
		selectedBg := theme.SelectBg
		selectedFg := theme.SelectFg
		if selectedBg == theme.PanelBg {
			selectedBg = theme.Accent
			selectedFg = theme.Bg
		}
		modal.SetSelectedBackgroundColor(selectedBg)
		modal.SetSelectedTextColor(selectedFg)
	}
	updateModalStyle()

	for _, t := range state.ThemeCatalog {
		modal.AddItem(t.Name, "", 0, nil)
	}

	modal.SetCurrentItem(state.ThemeIndex)

	// Preview theme when selection changes
	modal.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		state.ThemeIndex = index
		state.applyTheme(state.ThemeCatalog[index])
		state.updateFooter()
		state.updateHeaderMeta(state.LastUpdated, state.LastLoadErr)
		updateModalStyle()
	})

	closeModal := func(save bool) {
		if !save {
			// Restore original theme
			state.ThemeIndex = originalThemeIndex
			state.applyTheme(state.ThemeCatalog[originalThemeIndex])
			state.updateFooter()
			state.updateHeaderMeta(state.LastUpdated, state.LastLoadErr)
		} else {
			// Save theme to config
			state.saveAppConfig()
		}
		// Re-enable mouse and close modal
		state.App.EnableMouse(true)
		state.Pages.RemovePage("theme-modal")
		state.ThemeModalOpen = false
		state.App.SetFocus(state.HostList)
	}

	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			closeModal(false)
			return nil
		case tcell.KeyEnter:
			closeModal(true)
			return nil
		}
		if event.Rune() == 'q' {
			closeModal(false)
			return nil
		}
		return event
	})

	// Create a bordered modal box that holds the list plus a divider and footer
	modalWidth := 70
	modalHeight := len(state.ThemeCatalog) + 6
	if modalHeight < 10 {
		modalHeight = 10
	}

	// Footer centered and muted (instructions moved from title)
	theme := state.currentTheme()
	themeFooter := tview.NewTextView()
	themeFooter.SetTextAlign(tview.AlignCenter)
	themeFooter.SetTextColor(theme.Muted)
	themeFooter.SetBackgroundColor(theme.PanelBg)
	themeFooter.SetText("↑/↓ preview  Enter confirm  Esc cancel")

	// Divider above footer
	dividerLength := modalWidth - 4
	themeDivider := tview.NewTextView()
	themeDivider.SetTextAlign(tview.AlignCenter)
	themeDivider.SetTextColor(theme.Border)
	themeDivider.SetBackgroundColor(theme.PanelBg)
	themeDivider.SetText(strings.Repeat("─", dividerLength))

	// Bordered container for the modal list + footer
	modalBox := tview.NewFlex().SetDirection(tview.FlexRow)
	modalBox.SetBorder(true)
	modalBox.SetTitle(" Select Theme ")
	modalBox.SetTitleAlign(tview.AlignCenter)
	modalBox.SetBackgroundColor(theme.PanelBg)
	modalBox.SetBorderColor(theme.Border)
	modalBox.SetTitleColor(theme.Text)

	// Add list, small padding, divider, and footer inside the boxed modal
	paddingMid := tview.NewTextView()
	paddingMid.SetBackgroundColor(theme.PanelBg)
	modalBox.AddItem(modal, 0, 1, true)
	modalBox.AddItem(paddingMid, 1, 0, false)
	modalBox.AddItem(themeDivider, 1, 0, false)
	modalBox.AddItem(themeFooter, 1, 0, false)

	modalFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(modalBox, modalWidth, 0, true).
			AddItem(nil, 0, 1, false), modalHeight, 0, true).
		AddItem(nil, 0, 1, false)

	state.Pages.AddPage("theme-modal", modalFlex, true, true)
	state.App.SetFocus(modal)
}

func (state *AppState) showHelpModal() {
	state.ThemeModalOpen = true

	theme := state.currentTheme()

	// Disable mouse while help modal is open to block background interaction
	state.App.EnableMouse(false)

	// Build rune-aware padding for chips
	padCenter := func(s string, w int) string {
		rl := utf8.RuneCountInString(s)
		if rl >= w {
			return s
		}
		total := w - rl
		left := total / 2
		right := total - left
		return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
	}

	// Key chip builder
	chipWidth := 10
	makeKeyCell := func(text string) *tview.TableCell {
		display := padCenter(text, chipWidth)
		c := tview.NewTableCell(display)
		c.SetTextColor(theme.SelectFg)
		c.SetBackgroundColor(theme.SelectBg)
		c.SetAlign(tview.AlignCenter)
		c.SetSelectable(false)
		c.SetAttributes(tcell.AttrBold)
		c.SetMaxWidth(chipWidth)
		return c
	}

	makeDescCell := func(text string) *tview.TableCell {
		c := tview.NewTableCell(text)
		c.SetTextColor(theme.Text)
		// Center-align description and allow it to expand to fill column
		c.SetAlign(tview.AlignCenter)
		c.SetSelectable(false)
		c.SetExpansion(1)
		return c
	}

	// Left and right tables (each: key chip + description)
	leftTable := tview.NewTable()
	leftTable.SetSelectable(false, false)
	leftTable.SetBorders(false)
	leftTable.SetBackgroundColor(theme.PanelBg)

	rightTable := tview.NewTable()
	rightTable.SetSelectable(false, false)
	rightTable.SetBorders(false)
	rightTable.SetBackgroundColor(theme.PanelBg)

	// Content rows (unchanged texts)
	navRows := [][2]string{{"↑/↓", "move"}, {":", "search focus"}, {"Esc", "close"}}
	actRows := [][2]string{{"Enter", "connect"}, {"t", "theme"}, {"Ctrl+C", "quit"}, {"?", "help"}}

	// Add small header TextViews above each table (Navigation / Actions)
	navHeaderTV := tview.NewTextView()
	navHeaderTV.SetDynamicColors(true)
	navHeaderTV.SetTextAlign(tview.AlignCenter)
	navHeaderTV.SetText(fmt.Sprintf("[::b][%s]Navigation[-:-:-]", state.currentTheme().MarkupAccent))
	navHeaderTV.SetBackgroundColor(theme.PanelBg)

	actHeaderTV := tview.NewTextView()
	actHeaderTV.SetDynamicColors(true)
	actHeaderTV.SetTextAlign(tview.AlignCenter)
	actHeaderTV.SetText(fmt.Sprintf("[::b][%s]Actions[-:-:-]", state.currentTheme().MarkupAccent))
	actHeaderTV.SetBackgroundColor(theme.PanelBg)

	modalWidth := 70
	dividerLength := modalWidth - 4

	// Build per-column vertical containers (tables only); headers will be in a shared header row
	leftCol := tview.NewFlex().SetDirection(tview.FlexRow)
	leftCol.AddItem(leftTable, 0, 1, false)

	rightCol := tview.NewFlex().SetDirection(tview.FlexRow)
	rightCol.AddItem(rightTable, 0, 1, false)

	// Populate content rows starting at row 0
	maxRows := len(navRows)
	if len(actRows) > maxRows {
		maxRows = len(actRows)
	}
	for i := 0; i < maxRows; i++ {
		rowIndex := i
		if i < len(navRows) {
			leftTable.SetCell(rowIndex, 0, makeKeyCell(navRows[i][0]))
			leftTable.SetCell(rowIndex, 1, makeDescCell(navRows[i][1]))
		} else {
			leftTable.SetCell(rowIndex, 0, tview.NewTableCell(" ").SetBackgroundColor(theme.PanelBg))
			leftTable.SetCell(rowIndex, 1, tview.NewTableCell(" ").SetBackgroundColor(theme.PanelBg))
		}

		if i < len(actRows) {
			rightTable.SetCell(rowIndex, 0, makeKeyCell(actRows[i][0]))
			rightTable.SetCell(rowIndex, 1, makeDescCell(actRows[i][1]))
		} else {
			rightTable.SetCell(rowIndex, 0, tview.NewTableCell(" ").SetBackgroundColor(theme.PanelBg))
			rightTable.SetCell(rowIndex, 1, tview.NewTableCell(" ").SetBackgroundColor(theme.PanelBg))
		}
	}

	// Footer centered and muted
	footerTV := tview.NewTextView()
	footerTV.SetTextAlign(tview.AlignCenter)
	footerTV.SetTextColor(theme.Muted)
	footerTV.SetBackgroundColor(theme.PanelBg)
	footerTV.SetText("Esc to close")

	// Divider above footer as its own full-width TextView
	dividerTV := tview.NewTextView()
	dividerTV.SetTextAlign(tview.AlignCenter)
	dividerTV.SetTextColor(theme.Border)
	dividerTV.SetBackgroundColor(theme.PanelBg)
	dividerTV.SetText(strings.Repeat("─", dividerLength))

	// Shared header row for both columns (Navigation / Actions)
	headerRow := tview.NewFlex().SetDirection(tview.FlexColumn)
	headerRow.AddItem(navHeaderTV, 0, 1, false)
	headerRow.AddItem(actHeaderTV, 0, 1, false)

	// Single full-width divider under headers (matches footer divider length)
	headerDivider := tview.NewTextView()
	headerDivider.SetTextAlign(tview.AlignCenter)
	headerDivider.SetTextColor(theme.Border)
	headerDivider.SetBackgroundColor(theme.PanelBg)
	headerDivider.SetText(strings.Repeat("─", dividerLength))

	// Wrap in modal box with title 'Key Bindings'
	modalBox := tview.NewFlex().SetDirection(tview.FlexRow)
	modalBox.SetBorder(true)
	modalBox.SetTitle(" Key Bindings ")
	modalBox.SetTitleAlign(tview.AlignCenter)
	modalBox.SetBackgroundColor(theme.PanelBg)
	modalBox.SetBorderColor(theme.Border)
	modalBox.SetTitleColor(theme.Text)

	// Columns container (50:50) with a small gutter
	columns := tview.NewFlex().SetDirection(tview.FlexColumn)
	// Two equal-width columns (no visible gutter between them)
	columns.AddItem(leftCol, 0, 1, false)
	columns.AddItem(rightCol, 0, 1, false)

	// Assemble modal: header row, header divider, columns, divider, footer (reduced top padding)
	modalBox.AddItem(headerRow, 1, 0, false)
	modalBox.AddItem(headerDivider, 1, 0, false)
	// small breathing space between headers and content
	paddingMid := tview.NewTextView()
	paddingMid.SetBackgroundColor(theme.PanelBg)
	modalBox.AddItem(columns, 0, 1, true)
	modalBox.AddItem(paddingMid, 1, 0, false)
	modalBox.AddItem(dividerTV, 1, 0, false)
	modalBox.AddItem(footerTV, 1, 0, false)

	closeModal := func() {
		// Re-enable mouse and close modal
		state.App.EnableMouse(true)
		state.Pages.RemovePage("help-modal")
		state.ThemeModalOpen = false
		state.App.SetFocus(state.HostList)
	}

	modalBox.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			closeModal()
			return nil
		}
		return event
	})

	// Center modal on screen
	modalHeight := 6 + maxRows
	modalFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(modalBox, modalWidth, 0, true).
			AddItem(nil, 0, 1, false), modalHeight, 0, true).
		AddItem(nil, 0, 1, false)

	state.Pages.AddPage("help-modal", modalFlex, true, true)
	state.App.SetFocus(modalBox)
}

func (state *AppState) connectSSH() {
	if state.CurrentIndex < 0 || state.CurrentIndex >= len(state.Filtered) {
		return
	}

	entry := state.Filtered[state.CurrentIndex]
	if len(entry.Patterns) == 0 {
		return
	}

	// Use the first pattern as the host alias for ssh command
	host := entry.Patterns[0]

	// Stop the TUI application
	state.App.Stop()

	// Find ssh binary path
	sshPath, err := exec.LookPath("ssh")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ssh not found: %v\n", err)
		os.Exit(1)
	}

	// Replace current process with ssh using syscall.Exec
	// This gives full control to ssh including TTY handling
	if err := syscall.Exec(sshPath, []string{"ssh", host}, os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "failed to exec ssh: %v\n", err)
		os.Exit(1)
	}
}

func (state *AppState) applyTheme(theme AppTheme) {
	tview.Styles = theme.TView
	state.Root.SetBackgroundColor(theme.Bg)
	state.Header.SetBorderColor(theme.Border)
	state.Header.SetBackgroundColor(theme.HeaderBg)
	state.HeaderLogo.SetTextColor(theme.Accent)
	state.HeaderLogo.SetBackgroundColor(theme.HeaderBg)
	state.HeaderMeta.SetTextColor(theme.Text)
	state.HeaderMeta.SetBackgroundColor(theme.HeaderBg)

	state.Footer.SetBorderColor(theme.Border)
	state.Footer.SetTextColor(theme.Text)
	state.Footer.SetBackgroundColor(theme.FooterBg)
	state.updateFooter()

	state.SearchInput.SetLabelStyle(tcell.StyleDefault.Foreground(theme.Accent).Background(theme.PanelBg))
	state.SearchInput.SetFieldStyle(tcell.StyleDefault.Foreground(theme.Text).Background(theme.PanelBg))
	state.SearchInput.SetPlaceholderStyle(tcell.StyleDefault.Foreground(theme.Muted).Background(theme.PanelBg))
	state.SearchInput.SetBackgroundColor(theme.PanelBg)
	state.SearchInput.SetBorderColor(theme.Border)
	state.SearchInput.SetTitleColor(theme.Accent)

	state.HostList.SetBorderColor(theme.Border)
	state.HostList.SetMainTextStyle(tcell.StyleDefault.Foreground(theme.Text).Background(theme.PanelBg))
	state.HostList.SetSecondaryTextColor(theme.Muted)
	state.HostList.SetSelectedBackgroundColor(theme.Accent)
	state.HostList.SetSelectedTextColor(theme.Bg)
	state.HostList.SetBackgroundColor(theme.PanelBg)

	state.DetailTable.SetBorderColor(theme.Border)
	state.DetailTable.SetTitleColor(theme.Accent)
	state.DetailTable.SetBackgroundColor(theme.PanelBg)
}

func (state *AppState) updateFooter() {
	footer := fmt.Sprintf("[::b][%s]Ctrl+C[-:-:-] quit  [%s]:[-:-:-] search  [%s]t[-:-:-] theme  [%s]↑/↓[-:-:-] navigate  [%s]enter[-:-:-] connect  [%s]?[-:-:-] help",
		state.currentTheme().MarkupSuccess,
		state.currentTheme().MarkupAccent,
		state.currentTheme().MarkupAccent,
		state.currentTheme().MarkupAccent,
		state.currentTheme().MarkupSuccess,
		state.currentTheme().MarkupAccent,
	)
	state.Footer.SetText(footer)
}

func (state *AppState) currentTheme() AppTheme {
	return state.ThemeCatalog[state.ThemeIndex]
}

func getAppConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	configDir := filepath.Join(home, ".config", "55h")
	return filepath.Join(configDir, "config.yml")
}

func (state *AppState) loadAppConfig() {
	configPath := getAppConfigPath()
	if configPath == "" {
		return
	}

	// Try multiple candidate files for backward compatibility:
	//  - config.yml (new)
	//  - config (old)
	//  - config.json (legacy JSON)
	configDir := filepath.Dir(configPath)
	candidates := []string{
		configPath,
		filepath.Join(configDir, "config"),
		filepath.Join(configDir, "config.json"),
	}

	var data []byte
	var err error
	var used string
	for _, p := range candidates {
		data, err = os.ReadFile(p)
		if err == nil {
			used = p
			break
		}
	}
	if err != nil || len(data) == 0 {
		return
	}

	// Attempt JSON if file looks like JSON or has .json extension
	trimmed := strings.TrimSpace(string(data))
	var found string
	if strings.HasSuffix(used, ".json") || strings.HasPrefix(trimmed, "{") {
		var obj map[string]interface{}
		if jerr := json.Unmarshal(data, &obj); jerr == nil {
			// support keys: theme, theme_name
			if v, ok := obj["theme"]; ok {
				if s, ok := v.(string); ok {
					found = s
				}
			}
			if found == "" {
				if v, ok := obj["theme_name"]; ok {
					if s, ok := v.(string); ok {
						found = s
					}
				}
			}
		}
	} else {
		// Tiny key/value parser: look for lines like `theme: Value`, ignore blank/comment lines
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			if strings.HasPrefix(line, "#") {
				// allow comments, skip
				continue
			}
			// split on first ':'
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.ToLower(strings.TrimSpace(parts[0]))
			val := strings.TrimSpace(parts[1])
			if key == "theme" {
				found = val
				break
			}
		}
	}

	if found == "" {
		return
	}

	// Find theme by name (case-sensitive match as names are defined)
	for i, theme := range state.ThemeCatalog {
		if theme.Name == found {
			state.ThemeIndex = i
			return
		}
	}
	// if not found, leave ThemeIndex as-is (fallback)
}

func (state *AppState) saveAppConfig() {
	configPath := getAppConfigPath()
	if configPath == "" {
		return
	}

	// Ensure config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return
	}

	// Write simple key/value config with an English comment describing the setting
	// Format:
	// # Theme name for UI colors
	// theme: <Name>
	content := fmt.Sprintf("# Theme name for UI colors\ntheme: %s\n", state.currentTheme().Name)
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		// If we failed to write the new config, do not remove legacy files.
		return
	}

	// After successfully writing config.yml, attempt to remove legacy files
	// from the same config directory. Ignore any errors from removal.
	_ = os.Remove(filepath.Join(configDir, "config"))
	_ = os.Remove(filepath.Join(configDir, "config.json"))
}

func shortenPath(path string, max int) string {
	if len(path) <= max {
		return path
	}
	if max <= 3 {
		return path[:max]
	}
	return "..." + path[len(path)-(max-3):]
}

func fuzzyMatch(query string, target string) bool {
	if strings.TrimSpace(query) == "" {
		return true
	}
	needle := []rune(strings.ToLower(query))
	if len(needle) == 0 {
		return true
	}
	index := 0
	for _, r := range target {
		if unicode.ToLower(r) == needle[index] {
			index++
			if index == len(needle) {
				return true
			}
		}
	}
	return false
}

func parseBoolVal(s string) (*bool, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "yes", "true", "1":
		b := true
		return &b, true
	case "no", "false", "0":
		b := false
		return &b, true
	default:
		return nil, false
	}
}

func parseIntVal(s string) (*int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, false
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return nil, false
	}
	return &v, true
}

func loadSSHConfig(path string) ([]HostEntry, error) {
	if path == "" {
		return nil, fmt.Errorf("missing config path")
	}

	// Verify base file exists (top-level should error if missing)
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	visited := make(map[string]bool)
	entries := []HostEntry{}

	var loadFile func(string) error
	loadFile = func(p string) error {
		abs, err := filepath.Abs(p)
		if err == nil {
			p = abs
		}
		if visited[p] {
			return nil
		}
		visited[p] = true

		f, err := os.Open(p)
		if err != nil {
			// If top-level caller provided a path we already checked it exists.
			// For includes, callers should check existence before calling.
			return err
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		var current *HostEntry
		flush := func() {
			if current == nil {
				return
			}
			entries = append(entries, *current)
			current = nil
		}

		dir := filepath.Dir(p)

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) == 0 {
				continue
			}
			key := strings.ToLower(fields[0])

			if key == "include" {
				// Expand include patterns (supports multiple patterns on one line)
				for _, pat := range fields[1:] {
					// Resolve relative paths against current file dir
					if !filepath.IsAbs(pat) {
						pat = filepath.Join(dir, pat)
					}
					// Glob expansion
					matches, gerr := filepath.Glob(pat)
					if gerr != nil || len(matches) == 0 {
						// If no glob matches and the pattern is a plain file, check existence
						if strings.IndexAny(pat, "*?[]") == -1 {
							if _, sterr := os.Stat(pat); sterr == nil {
								// single file exists
								_ = loadFile(pat)
							}
						}
						continue
					}
					for _, m := range matches {
						// Ignore missing files quietly
						if _, statErr := os.Stat(m); statErr != nil {
							continue
						}
						_ = loadFile(m)
					}
				}
				continue
			}

			if key == "host" {
				flush()
				if len(fields) > 1 {
					current = &HostEntry{Patterns: fields[1:]}
				} else {
					current = &HostEntry{Patterns: []string{}}
				}
				continue
			}

			if current == nil {
				continue
			}

			value := strings.TrimSpace(strings.Join(fields[1:], " "))
			switch key {
			case "hostname":
				current.HostName = value
			case "user":
				current.User = value
			case "port":
				current.Port = value
			case "identityfile":
				current.IdentityFile = value
			case "proxyjump":
				current.ProxyJump = value
			case "serveraliveinterval":
				if v, ok := parseIntVal(value); ok {
					current.ServerAliveInterval = v
				}
			case "serveralivecountmax":
				if v, ok := parseIntVal(value); ok {
					current.ServerAliveCountMax = v
				}
			case "forwardagent":
				if b, ok := parseBoolVal(value); ok {
					current.ForwardAgent = b
				}
			case "identitiesonly":
				if b, ok := parseBoolVal(value); ok {
					current.IdentitiesOnly = b
				}
			}
		}

		if err := scanner.Err(); err != nil {
			return err
		}

		flush()
		return nil
	}

	if err := loadFile(path); err != nil {
		return nil, err
	}

	return entries, nil
}

func formatBoolYesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// handleAddSSH implements: 55h add ssh user@host [-p port] [-i identity] [-J jump] [-o Key=Value ...] [--name alias]
func handleAddSSH(args []string, configPath string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: 55h add ssh user@host [-p port] [-i identity] [-J jump] [-o Key=Value ...] [--name alias]")
	}

	target := args[0]
	var port, identity, jump, name string
	var serverAliveInterval *int
	var serverAliveCountMax *int
	var forwardAgent *bool
	var identitiesOnly *bool

	for i := 1; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-p":
			if i+1 >= len(args) {
				return fmt.Errorf("-p requires a value")
			}
			port = args[i+1]
			i++
		case "-i":
			if i+1 >= len(args) {
				return fmt.Errorf("-i requires a value")
			}
			identity = args[i+1]
			i++
		case "-J":
			if i+1 >= len(args) {
				return fmt.Errorf("-J requires a value")
			}
			jump = args[i+1]
			i++
		case "-o":
			if i+1 >= len(args) {
				return fmt.Errorf("-o requires Key=Value")
			}
			kv := args[i+1]
			i++
			parts := strings.SplitN(kv, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.ToLower(strings.TrimSpace(parts[0]))
			val := strings.TrimSpace(parts[1])
			switch key {
			case "forwardagent":
				if b, ok := parseBoolVal(val); ok {
					forwardAgent = b
				}
			case "identitiesonly":
				if b, ok := parseBoolVal(val); ok {
					identitiesOnly = b
				}
			case "serveraliveinterval":
				if v, ok := parseIntVal(val); ok {
					serverAliveInterval = v
				}
			case "serveralivecountmax":
				if v, ok := parseIntVal(val); ok {
					serverAliveCountMax = v
				}
			default:
				// ignore unknown -o keys
			}
		case "--name":
			if i+1 >= len(args) {
				return fmt.Errorf("--name requires a value")
			}
			name = args[i+1]
			i++
		default:
			return fmt.Errorf("unknown argument: %s", a)
		}
	}

	// parse target user@host
	var user, host string
	if strings.Contains(target, "@") {
		parts := strings.SplitN(target, "@", 2)
		user = parts[0]
		host = parts[1]
	} else {
		host = target
	}

	// Determine alias name: prompt if TTY and not provided, else require --name
	fi, err := os.Stdin.Stat()
	isTTY := err == nil && (fi.Mode()&os.ModeCharDevice) != 0

	// Derive default alias from target: user@host if user provided, else host
	defaultAlias := host
	if user != "" && host != "" {
		defaultAlias = fmt.Sprintf("%s@%s", user, host)
	}

	if name == "" {
		if isTTY {
			// Prompt showing suggestion; if user presses Enter, use default
			if defaultAlias != "" {
				// Show the default suggestion in grey when running in a TTY
				// Display it as a placeholder-style value after the label.
				// Use ANSI bright-black (90) for grey and reset after.
				fmt.Printf("Alias: \x1b[90m%s\x1b[0m ", defaultAlias)
			} else {
				fmt.Print("Alias: ")
			}
			reader := bufio.NewReader(os.Stdin)
			line, _ := reader.ReadString('\n')
			entered := strings.TrimSpace(line)
			if entered == "" {
				if defaultAlias == "" {
					return fmt.Errorf("alias required")
				}
				name = defaultAlias
			} else {
				name = entered
			}
		} else {
			return fmt.Errorf("--name is required when stdin is not a TTY")
		}
	}

	// Ensure parent dir exists
	cfg := configPath
	if cfg == "" {
		return fmt.Errorf("unable to resolve config path")
	}
	if err := os.MkdirAll(filepath.Dir(cfg), 0755); err != nil {
		return fmt.Errorf("failed to create parent dir: %v", err)
	}

	// Check for duplicate alias in existing config (including includes)
	if entries, lerr := loadSSHConfig(cfg); lerr == nil {
		for _, e := range entries {
			for _, p := range e.Patterns {
				if p == name {
					return fmt.Errorf("alias %s already exists in %s", name, cfg)
				}
			}
		}
	}

	// Build host block
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("Host %s\n", name))
	if host != "" {
		sb.WriteString(fmt.Sprintf("    HostName %s\n", host))
	}
	if user != "" {
		sb.WriteString(fmt.Sprintf("    User %s\n", user))
	}
	if port != "" {
		sb.WriteString(fmt.Sprintf("    Port %s\n", port))
	}
	if identity != "" {
		sb.WriteString(fmt.Sprintf("    IdentityFile %s\n", identity))
	}
	if jump != "" {
		sb.WriteString(fmt.Sprintf("    ProxyJump %s\n", jump))
	}
	if forwardAgent != nil {
		sb.WriteString(fmt.Sprintf("    ForwardAgent %s\n", formatBoolYesNo(*forwardAgent)))
	}
	if identitiesOnly != nil {
		sb.WriteString(fmt.Sprintf("    IdentitiesOnly %s\n", formatBoolYesNo(*identitiesOnly)))
	}
	if serverAliveInterval != nil {
		sb.WriteString(fmt.Sprintf("    ServerAliveInterval %d\n", *serverAliveInterval))
	}
	if serverAliveCountMax != nil {
		sb.WriteString(fmt.Sprintf("    ServerAliveCountMax %d\n", *serverAliveCountMax))
	}

	// Append to file
	f, err := os.OpenFile(cfg, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open config file: %v", err)
	}
	defer f.Close()

	if _, err := f.WriteString(sb.String()); err != nil {
		return fmt.Errorf("failed to write config: %v", err)
	}

	return nil
}
