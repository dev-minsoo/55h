package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
	"unicode"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type HostEntry struct {
	Patterns     []string
	HostName     string
	User         string
	Port         string
	IdentityFile string
	ProxyJump    string
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
		case 'q':
			app.Stop()
			return nil
		case 't':
			state.showThemeModal()
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
	originalThemeIndex := state.ThemeIndex

	modal := tview.NewList()
	modal.SetBorder(true)
	modal.SetTitle(" Select Theme (↑/↓ preview, Enter confirm, Esc cancel) ")
	modal.SetTitleAlign(tview.AlignCenter)
	modal.ShowSecondaryText(false)
	modal.SetHighlightFullLine(true)

	// Style the modal with current theme
	updateModalStyle := func() {
		theme := state.currentTheme()
		modal.SetBackgroundColor(theme.PanelBg)
		modal.SetBorderColor(theme.Accent)
		modal.SetTitleColor(theme.Accent)
		modal.SetMainTextStyle(tcell.StyleDefault.Foreground(theme.Text).Background(theme.PanelBg))
		modal.SetSelectedBackgroundColor(theme.Accent)
		modal.SetSelectedTextColor(theme.Bg)
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

	// Create centered modal container
	modalWidth := 50
	modalHeight := len(state.ThemeCatalog) + 2

	modalFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(modal, modalWidth, 0, true).
			AddItem(nil, 0, 1, false), modalHeight, 0, true).
		AddItem(nil, 0, 1, false)

	state.Pages.AddPage("theme-modal", modalFlex, true, true)
	state.App.SetFocus(modal)
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
	footer := fmt.Sprintf("[::b][%s]q[-:-:-] quit  [%s]:[-:-:-] search  [%s]t[-:-:-] theme  [%s]↑/↓[-:-:-] navigate  [%s]enter[-:-:-] connect",
		state.currentTheme().MarkupSuccess,
		state.currentTheme().MarkupAccent,
		state.currentTheme().MarkupAccent,
		state.currentTheme().MarkupAccent,
		state.currentTheme().MarkupSuccess,
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
	return filepath.Join(configDir, "config.json")
}

func (state *AppState) loadAppConfig() {
	configPath := getAppConfigPath()
	if configPath == "" {
		return
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return
	}

	var config AppConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return
	}

	// Find theme by name
	for i, theme := range state.ThemeCatalog {
		if theme.Name == config.ThemeName {
			state.ThemeIndex = i
			break
		}
	}
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

	config := AppConfig{
		ThemeName: state.currentTheme().Name,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return
	}

	_ = os.WriteFile(configPath, data, 0644)
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

func loadSSHConfig(path string) ([]HostEntry, error) {
	if path == "" {
		return nil, fmt.Errorf("missing config path")
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var entries []HostEntry
	var current *HostEntry

	flush := func() {
		if current == nil {
			return
		}
		entries = append(entries, *current)
		current = nil
	}

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
		if key == "host" {
			flush()
			current = &HostEntry{Patterns: fields[1:]}
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
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	flush()

	sort.Slice(entries, func(i, j int) bool {
		left := strings.Join(entries[i].Patterns, " ")
		right := strings.Join(entries[j].Patterns, " ")
		return left < right
	})

	return entries, nil
}
