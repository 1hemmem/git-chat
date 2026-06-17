package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"git-chat/internal/chat"
	"git-chat/internal/repo"
)

var (
	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8"))

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("14"))

	sepStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	inputBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Padding(0, 1)

	timeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	authorColors = []lipgloss.Color{"3", "4", "5", "6", "9", "10", "11", "13"}

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Italic(true)

	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("14")).
			Bold(true)
)

type messagesMsg struct {
	msgs []chat.Message
	err  error
}

type tickMsg time.Time

type sendResultMsg struct {
	body string
	err  error
}

type model struct {
	repoName  string
	repoFull  string
	username  string
	messages  []chat.Message
	sentOk    map[string]bool
	ready     bool
	err       error
	spinner   spinner.Model
	viewport  viewport.Model
	textinput textinput.Model
}

func initialModel(repoName string) (model, error) {
	ti := textinput.New()
	ti.Placeholder = "Type a message..."
	ti.Focus()
	ti.Prompt = ""

	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().Padding(0, 1)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))

	repoFull, err := repo.ResolveGroup(repoName)
	if err != nil {
		return model{}, err
	}

	return model{
		repoName:  repoName,
		repoFull:  repoFull,
		sentOk:    make(map[string]bool),
		spinner:   sp,
		textinput: ti,
		viewport:  vp,
	}, nil
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		fetchMessages(m.repoFull),
		fetchUsername(),
		tickCmd(),
		m.spinner.Tick,
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		contentWidth := msg.Width - 4
		headerHeight := 2
		footerHeight := 5
		m.viewport = viewport.New(contentWidth, msg.Height-headerHeight-footerHeight)
		m.viewport.Style = lipgloss.NewStyle().Padding(0, 1)
		m.viewport.SetContent(m.loadingView())
		m.textinput.Width = contentWidth - 2
		m.ready = true
		m.viewport.GotoBottom()

	case messagesMsg:
		if msg.err != nil {
			return m, nil
		}
		var pending []chat.Message
		for _, cm := range m.messages {
			if cm.Author == m.username {
				if _, ok := m.sentOk[cm.Body]; !ok {
					pending = append(pending, cm)
				}
			}
		}
		m.messages = msg.msgs
		for _, cm := range m.messages {
			if cm.Author == m.username {
				m.sentOk[cm.Body] = true
			}
		}
		for _, p := range pending {
			found := false
			for _, cm := range m.messages {
				if cm.Author == p.Author && cm.Body == p.Body {
					found = true
					break
				}
			}
			if !found {
				m.messages = append(m.messages, p)
			}
		}
		m.viewport.SetContent(m.viewportContent())
		m.viewport.GotoBottom()

	case usernameMsg:
		if msg.err != nil {
			return m, nil
		}
		m.username = msg.username

	case tickMsg:
		return m, tea.Batch(fetchMessages(m.repoFull), tickCmd())

	case sendResultMsg:
		if msg.err != nil {
			m.err = msg.err
			delete(m.sentOk, msg.body)
			var kept []chat.Message
			for _, cm := range m.messages {
				if !(cm.Author == m.username && cm.Body == msg.body) {
					kept = append(kept, cm)
				}
			}
			m.messages = kept
			m.viewport.SetContent(m.viewportContent())
			m.viewport.GotoBottom()
			return m, nil
		}
		m.sentOk[msg.body] = true
		m.viewport.SetContent(m.viewportContent())
		m.viewport.GotoBottom()
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEscape, tea.KeyCtrlC:
			return m, tea.Quit

		case tea.KeyEnter:
			body := strings.TrimSpace(m.textinput.Value())
			if body == "" {
				return m, nil
			}
			m.textinput.Reset()
			localMsg := chat.Message{
				Author:    m.username,
				Timestamp: time.Now().UTC().Format("20060102T150405Z"),
				Body:      body,
			}
			m.messages = append(m.messages, localMsg)
			m.viewport.SetContent(m.viewportContent())
			m.viewport.GotoBottom()
			return m, sendMessage(m.repoFull, body)
		}

	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		if cmd != nil {
			return m, cmd
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	m.textinput, _ = m.textinput.Update(msg)

	return m, cmd
}

func (m model) View() string {
	if !m.ready {
		return "Loading...\n"
	}
	if m.err != nil {
		return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("1")).Render(fmt.Sprintf(" Error: %v ", m.err))
	}

	title := titleStyle.Render(" Git Chat ") + sepStyle.Render("·") + titleStyle.Render(" "+m.repoFull+" ")

	status := fmt.Sprintf(" %d messages  │  Enter: send  │  Esc: quit ", len(m.messages))
	if len(m.messages) == 0 {
		status = "  No messages yet  │  Enter: send  │  Esc: quit "
	}
	status = statusStyle.Render(status)

	v := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		m.viewport.View(),
		inputBoxStyle.Render(promptStyle.Render("✎  ")+" "+m.textinput.View()),
		status,
	)

	return borderStyle.Render(v)
}

func (m model) loadingView() string {
	s := m.spinner.View() + " Loading messages..."
	return lipgloss.NewStyle().Padding(1).Render(s)
}

func (m model) viewportContent() string {
	if len(m.messages) == 0 {
		return lipgloss.NewStyle().Padding(1).Foreground(lipgloss.Color("8")).Render("No messages yet.")
	}
	var lines []string
	authorStyles := make(map[string]lipgloss.Style)
	colorIdx := 0
	for _, msg := range m.messages {
		if _, ok := authorStyles[msg.Author]; !ok {
			authorStyles[msg.Author] = lipgloss.NewStyle().
				Bold(true).
				Foreground(authorColors[colorIdx%len(authorColors)])
			colorIdx++
		}
		aStyle := authorStyles[msg.Author]

		t, err := time.Parse("20060102T150405Z", msg.Timestamp)
		displayTime := msg.Timestamp
		if err == nil {
			displayTime = t.Local().Format("15:04")
		}

		prefix := "  "
		if msg.Author == m.username {
			delivered, exists := m.sentOk[msg.Body]
			if !exists {
				prefix = "~ "
			} else if delivered {
				prefix = promptStyle.Bold(true).Render("✓") + " "
			} else {
				prefix = "  "
			}
		}

		line := timeStyle.Render(displayTime) + " " + prefix + aStyle.Render(msg.Author) + " " + msg.Body
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

type usernameMsg struct {
	username string
	err      error
}

func fetchUsername() tea.Cmd {
	return func() tea.Msg {
		username, err := repo.GetGitHubUsername()
		return usernameMsg{username, err}
	}
}

func fetchMessages(repoFull string) tea.Cmd {
	return func() tea.Msg {
		localPath := repo.CachePath(repoFull)
		repo.PullIfNew(repoFull, localPath)
		msgs, err := chat.ReadMessagesFromCache(localPath)
		return messagesMsg{msgs, err}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func sendMessage(repoFull, body string) tea.Cmd {
	return func() tea.Msg {
		err := chat.SendMessage(repoFull, body)
		return sendResultMsg{body: body, err: err}
	}
}

func Run(repoName string) error {
	m, err := initialModel(repoName)
	if err != nil {
		return err
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
