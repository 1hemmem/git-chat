package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"git-chat/internal/chat"
	"git-chat/internal/repo"
)

type messagesMsg struct {
	msgs []chat.Message
	err  error
}

type tickMsg time.Time

type sendResultMsg struct {
	err error
}

type model struct {
	repoName  string
	repoFull  string
	username  string
	messages  []chat.Message
	ready     bool
	err       error
	viewport  viewport.Model
	textinput textinput.Model
}

func initialModel(repoName string) model {
	ti := textinput.New()
	ti.Placeholder = "Type a message..."
	ti.Focus()

	vp := viewport.New(80, 20)

	repoFull := repo.ResolveRepo(repoName)

	return model{
		repoName:  repoName,
		repoFull:  repoFull,
		textinput: ti,
		viewport:  vp,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		fetchMessages(m.repoName),
		fetchUsername(),
		tickCmd(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		headerHeight := 3
		footerHeight := 3
		m.viewport = viewport.New(msg.Width, msg.Height-headerHeight-footerHeight)
		m.viewport.SetContent(m.viewportContent())
		m.textinput.Width = msg.Width
		m.ready = true
		m.viewport.GotoBottom()

	case messagesMsg:
		if msg.err != nil {
			return m, nil
		}
		m.messages = msg.msgs
		m.viewport.SetContent(m.viewportContent())
		m.viewport.GotoBottom()

	case usernameMsg:
		if msg.err != nil {
			return m, nil
		}
		m.username = msg.username

	case tickMsg:
		return m, fetchMessages(m.repoName)

	case sendResultMsg:
		if msg.err != nil {
			m.err = msg.err
		}
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
			return m, sendMessage(m.repoName, body)
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
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")).
		Padding(0, 1).
		Render("Git Chat — " + m.repoFull)

	sep := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(lipgloss.Color("8")).
		Render(strings.Repeat(" ", m.viewport.Width))

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		sep,
		m.viewport.View(),
		sep,
		m.textinput.View(),
	)
}

func (m model) viewportContent() string {
	if len(m.messages) == 0 {
		return "No messages yet."
	}
	var lines []string
	for _, msg := range m.messages {
		t, err := time.Parse("20060102T150405Z", msg.Timestamp)
		displayTime := msg.Timestamp
		if err == nil {
			displayTime = t.Local().Format("2006-01-02 15:04")
		}
		lines = append(lines, "["+displayTime+"] "+msg.Author+": "+msg.Body)
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

func fetchMessages(repoName string) tea.Cmd {
	return func() tea.Msg {
		msgs, err := chat.ReadMessages(repoName)
		return messagesMsg{msgs, err}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func sendMessage(repoName, body string) tea.Cmd {
	return func() tea.Msg {
		err := chat.SendMessage(repoName, body)
		return sendResultMsg{err}
	}
}

func Run(repoName string) error {
	p := tea.NewProgram(initialModel(repoName), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
