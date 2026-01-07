package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/gm-agent-org/gm-agent/packages/cli/internal/client"
	"github.com/spf13/viper"
)

type errMsg error

type model struct {
	client    *client.Client
	waiting   bool
	sessionID string
	messages  []string // Rendered markdown strings
	viewport  viewport.Model
	textarea  textarea.Model
	spinner   spinner.Model
	renderer  *glamour.TermRenderer
	err       error

	// Streaming state
	streamingInProgress bool
	streamingRaw        string

	eventCh     <-chan client.Event
	ctx         context.Context
	cancel      context.CancelFunc
	initialized bool
}

// Custom Messages
type sessionCreatedMsg string
type eventMsg client.Event
type nextEventMsg struct{}

func initialModel(cfg *Config) model {
	ta := textarea.New()
	ta.Placeholder = "Send a message... (Type /new to reset, /exit to quit)"
	ta.Focus()
	ta.Prompt = "┃ "
	ta.CharLimit = 0 // Unlimited
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false) // Enter sends message

	vp := viewport.New(80, 20)
	vp.SetContent("Initializing gm-agent...")

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// Initialize API Client
	apiKey := viper.GetString("api-key")
	if apiKey == "" {
		// Just a warning in the UI
	}
	c, err := client.New(cfg.Server, apiKey, cfg.Timeout)
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		os.Exit(1)
	}

	r, _ := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(80),
	)

	ctx, cancel := context.WithCancel(context.Background())

	return model{
		client:   c,
		textarea: ta,
		viewport: vp,
		spinner:  sp,
		ctx:      ctx,
		cancel:   cancel,
		renderer: r,
		messages: []string{},
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.spinner.Tick,
		createSessionCmd(m.client, m.ctx),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		spCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)
	m.spinner, spCmd = m.spinner.Update(msg)

	switch msg := msg.(type) {
	// Key Presses
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			if m.waiting {
				return m, nil
			}
			input := strings.TrimSpace(m.textarea.Value())
			if input == "" {
				return m, nil
			}

			// Handle Slash Commands
			if input == "/exit" || input == "/quit" {
				return m, tea.Quit
			}
			if input == "/new" {
				m.waiting = true
				m.textarea.Reset()
				m.messages = append(m.messages, styleSystemMessage("Starting new session..."))
				m.updateViewport()
				return m, createSessionCmd(m.client, m.ctx)
			}

			// Add User Message to History
			m.messages = append(m.messages, styleUserMessage(input, m.renderer))
			m.textarea.Reset()
			m.waiting = true
			m.updateViewport()

			// Send to API
			return m, sendMessageCmd(m.client, m.ctx, m.sessionID, input)
		}

	case errMsg:
		m.err = msg
		m.messages = append(m.messages, styleSystemMessage(fmt.Sprintf("Error: %v", msg)))
		m.waiting = false
		m.updateViewport()
		return m, nil

	case sessionCreatedMsg:
		m.sessionID = string(msg)
		m.waiting = false
		m.messages = append(m.messages, styleSystemMessage(fmt.Sprintf("Session Started (%s)", m.sessionID)))
		m.updateViewport()
		// Start Streaming Events interaction
		return m, streamEventsCmd(m.client, m.ctx, m.sessionID)

	case streamReadyMsg:
		m.eventCh = msg.ch
		return m, waitForNextEvent(m.eventCh)

	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.textarea.SetWidth(msg.Width)
		m.viewport.Height = msg.Height - m.textarea.Height() - 2 // Space for input
		m.updateViewport()
		// Update renderer wrap
		m.renderer, _ = glamour.NewTermRenderer(
			glamour.WithStandardStyle("dark"),
			glamour.WithWordWrap(msg.Width-4),
		)

	// Event Stream Messages
	case client.Event:
		m.waiting = false
		switch msg.Type {
		case "llm_token":
			var data struct {
				Delta string `json:"delta"`
			}
			if err := json.Unmarshal(msg.Data, &data); err == nil && data.Delta != "" {
				if !m.streamingInProgress {
					m.streamingInProgress = true
					m.streamingRaw = ""
					m.messages = append(m.messages, styleAssistantMessage(""))
				}
				m.streamingRaw += data.Delta
				m.messages[len(m.messages)-1] = styleAssistantMessage(m.streamingRaw)
			}
		case "llm_response":
			wasStreaming := m.streamingInProgress
			m.streamingInProgress = false
			m.streamingRaw = ""
			var data struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"tool_calls"`
			}
			if err := json.Unmarshal(msg.Data, &data); err == nil {
				if data.Content != "" {
					rendered, _ := m.renderer.Render(data.Content)
					if wasStreaming && len(m.messages) > 0 {
						// Replace the stream placeholder with final rendered content
						m.messages[len(m.messages)-1] = styleAssistantMessage(rendered)
					} else {
						m.messages = append(m.messages, styleAssistantMessage(rendered))
					}
				}
				// ... (tool calls etc)
				for _, tc := range data.ToolCalls {
					if tc.Name == "talk" {
						var args struct {
							Message string `json:"message"`
						}
						if err := json.Unmarshal([]byte(tc.Arguments), &args); err == nil {
							rendered, _ := m.renderer.Render(args.Message)
							m.messages = append(m.messages, styleAssistantMessage(rendered))
						}
					} else {
						m.messages = append(m.messages, styleSystemMessage(fmt.Sprintf("🛠️  Agent wants to use: %s", tc.Name)))
					}
				}
			}
		case "tool_result":
			var data struct {
				ToolName string `json:"tool_name"`
				Success  bool   `json:"success"`
				Error    string `json:"error"`
			}
			if err := json.Unmarshal(msg.Data, &data); err == nil {
				if data.ToolName != "talk" {
					status := "✅ success"
					if !data.Success {
						status = fmt.Sprintf("❌ failed: %s", data.Error)
					}
					m.messages = append(m.messages, styleSystemMessage(fmt.Sprintf("⚙️  Tool %s: %s", data.ToolName, status)))
				}
			}
		case "error":
			var data struct {
				Error string `json:"error"`
			}
			json.Unmarshal(msg.Data, &data)
			m.messages = append(m.messages, styleSystemMessage(fmt.Sprintf("🛑 Error: %s", data.Error)))
		}
		m.updateViewport()
		return m, waitForNextEvent(m.eventCh)

	}

	return m, tea.Batch(tiCmd, vpCmd, spCmd)
}

func (m model) View() string {
	var s strings.Builder

	// Header / Viewport
	s.WriteString(m.viewport.View())
	s.WriteString("\n")

	// Spinner / Status
	if m.waiting {
		s.WriteString(m.spinner.View() + " Thinking...\n")
	} else {
		s.WriteString("\n")
	}

	// Input
	s.WriteString(m.textarea.View())

	return s.String()
}

func (m *model) updateViewport() {
	m.viewport.SetContent(strings.Join(m.messages, "\n\n"))
	m.viewport.GotoBottom()
}

// Commands

func createSessionCmd(c *client.Client, ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		sid, err := c.CreateSession(ctx, "Ready to help. What would you like to do?")
		if err != nil {
			return errMsg(err)
		}
		return sessionCreatedMsg(sid)
	}
}

func sendMessageCmd(c *client.Client, ctx context.Context, sid, content string) tea.Cmd {
	return func() tea.Msg {
		if err := c.SendMessage(ctx, sid, content); err != nil {
			return errMsg(err)
		}
		// We don't return a specific msg, the event stream will yield responses
		return nil
	}
}

func streamEventsCmd(c *client.Client, ctx context.Context, sid string) tea.Cmd {
	return func() tea.Msg {
		ch, err := c.StreamEvents(ctx, sid)
		if err != nil {
			return errMsg(err)
		}
		// Return a specific msg to set the channel on the model
		// But update is pure... how to set channel?
		// We need a Msg type that carries the channel
		return streamReadyMsg{ch}
	}
}

type streamReadyMsg struct {
	ch <-chan client.Event
}

func waitForNextEvent(ch <-chan client.Event) tea.Cmd {
	return func() tea.Msg {
		if ch == nil {
			return nil
		}
		evt, ok := <-ch
		if !ok {
			return nil // Stream closed
		}
		return evt
	}
}

// Logic fix:
// Init -> createSession -> success(sid) -> streamEvents -> streamReady(ch).

// We need to add streamReadyMsg handling to Update.

// Styles

func styleUserMessage(msg string, r *glamour.TermRenderer) string {
	// return fmt.Sprintf("👤 **You**: %s", msg)
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Render("👤 You:") + "\n" + msg
}

func styleAssistantMessage(msg string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Render("🤖 Assistant:") + "\n" + msg
}

func styleSystemMessage(msg string) string {
	return lipgloss.NewStyle().
		Faint(true).
		Render(msg)
}

// Main Entry Point

func replLoop(cfg *Config) error {
	p := tea.NewProgram(initialModel(cfg), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		return err
	}
	return nil
}
