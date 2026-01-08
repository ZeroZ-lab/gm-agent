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

	// Permission state
	permissionRequest *struct {
		RequestID      string
		ToolName       string
		Permission     string
		Patterns       []string
		SelectedOption int // 0=Allow once, 1=Deny, 2=Always allow, 3=Deny all
	}

	// UI enhancements
	welcomeInfo   WelcomeInfo
	inputHistory  []string
	historyIndex  int
	showStatusBar bool
	width         int
	height        int
	model         string // LLM model name
}

// Custom Messages
type sessionCreatedMsg struct {
	sessionID      string
	pendingMessage string // Message to send after session creation
}
type eventMsg client.Event
type nextEventMsg struct{}
type permissionHandledMsg struct{}

func initialModel(cfg *Config) model {
	ta := textarea.New()
	ta.Placeholder = "Ask anything... (Shift+Enter for new line)"
	ta.Focus()
	ta.Prompt = "❯ "
	ta.CharLimit = 0 // Unlimited
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	// Enable multiline: Shift+Enter inserts newline, Enter sends
	ta.KeyMap.InsertNewline.SetKeys("shift+enter")

	vp := viewport.New(80, 20)

	// Get welcome info
	welcomeInfo := GetWelcomeInfo()
	vp.SetContent(RenderWelcome(welcomeInfo))

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B35"))

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
		client:        c,
		textarea:      ta,
		viewport:      vp,
		spinner:       sp,
		ctx:           ctx,
		cancel:        cancel,
		renderer:      r,
		messages:      []string{},
		welcomeInfo:   welcomeInfo,
		inputHistory:  []string{},
		historyIndex:  -1,
		showStatusBar: true,
		model:         "default",
	}
}

func (m model) Init() tea.Cmd {
	// Don't create session on init - wait for first user message
	return tea.Batch(
		textarea.Blink,
		m.spinner.Tick,
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		spCmd tea.Cmd
	)

	// Update components normally unless blocked by permission
	if m.permissionRequest == nil {
		m.textarea, tiCmd = m.textarea.Update(msg)
	}
	m.viewport, vpCmd = m.viewport.Update(msg)
	m.spinner, spCmd = m.spinner.Update(msg)

	switch msg := msg.(type) {
	// Key Presses
	case tea.KeyMsg:
		// Handle permission interaction
		if m.permissionRequest != nil {
			switch msg.String() {
			case "y", "Y":
				return m, submitPermissionCmd(m.client, m.ctx, m.sessionID, m.permissionRequest.RequestID, true, false)
			case "n", "N":
				return m, submitPermissionCmd(m.client, m.ctx, m.sessionID, m.permissionRequest.RequestID, false, false)
			case "a", "A":
				return m, submitPermissionCmd(m.client, m.ctx, m.sessionID, m.permissionRequest.RequestID, true, true)
			case "d", "D":
				// Deny all - deny with "always" flag (block future requests)
				return m, submitPermissionCmd(m.client, m.ctx, m.sessionID, m.permissionRequest.RequestID, false, true)
			case "up", "k":
				// Navigate up in options
				if m.permissionRequest.SelectedOption > 0 {
					m.permissionRequest.SelectedOption--
				}
				return m, nil
			case "down", "j":
				// Navigate down in options
				if m.permissionRequest.SelectedOption < 3 {
					m.permissionRequest.SelectedOption++
				}
				return m, nil
			case "enter":
				// Execute selected option
				switch m.permissionRequest.SelectedOption {
				case 0: // Allow once
					return m, submitPermissionCmd(m.client, m.ctx, m.sessionID, m.permissionRequest.RequestID, true, false)
				case 1: // Deny
					return m, submitPermissionCmd(m.client, m.ctx, m.sessionID, m.permissionRequest.RequestID, false, false)
				case 2: // Always allow
					return m, submitPermissionCmd(m.client, m.ctx, m.sessionID, m.permissionRequest.RequestID, true, true)
				case 3: // Deny all
					return m, submitPermissionCmd(m.client, m.ctx, m.sessionID, m.permissionRequest.RequestID, false, true)
				}
			}
			return m, nil // Ignore other keys while waiting for permission
		}

		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyUp:
			// History navigation: previous
			if len(m.inputHistory) > 0 && !m.waiting {
				if m.historyIndex < len(m.inputHistory)-1 {
					m.historyIndex++
					m.textarea.SetValue(m.inputHistory[len(m.inputHistory)-1-m.historyIndex])
				}
			}
			return m, nil
		case tea.KeyDown:
			// History navigation: next
			if m.historyIndex > 0 {
				m.historyIndex--
				m.textarea.SetValue(m.inputHistory[len(m.inputHistory)-1-m.historyIndex])
			} else if m.historyIndex == 0 {
				m.historyIndex = -1
				m.textarea.SetValue("")
			}
			return m, nil
		case tea.KeyEnter:
			if m.waiting {
				return m, nil
			}
			input := strings.TrimSpace(m.textarea.Value())
			if input == "" {
				return m, nil
			}

			// Save to history
			if len(m.inputHistory) == 0 || m.inputHistory[len(m.inputHistory)-1] != input {
				m.inputHistory = append(m.inputHistory, input)
			}
			m.historyIndex = -1

			// Handle Slash Commands
			switch {
			case input == "/exit" || input == "/quit":
				return m, tea.Quit
			case input == "/new":
				m.waiting = true
				m.textarea.Reset()
				m.messages = []string{}
				m.messages = append(m.messages, styleSystemMessage("Starting new session..."))
				m.updateViewport()
				return m, createSessionCmd(m.client, m.ctx)
			case input == "/clear":
				m.textarea.Reset()
				m.messages = []string{}
				m.updateViewport()
				return m, nil
			case input == "/help":
				m.textarea.Reset()
				m.messages = append(m.messages, RenderHelp())
				m.updateViewport()
				return m, nil
			case input == "/history":
				m.textarea.Reset()
				historyMsg := styleSystemMsg.Render("Input History:\n")
				for i, h := range m.inputHistory {
					historyMsg += fmt.Sprintf("  %d. %s\n", i+1, h)
				}
				m.messages = append(m.messages, historyMsg)
				m.updateViewport()
				return m, nil
			case input == "/checkpoints":
				if m.sessionID == "" {
					m.messages = append(m.messages, styleSystemMessage("⚠️ No active session"))
					m.updateViewport()
					return m, nil
				}
				m.textarea.Reset()
				m.waiting = true
				return m, listCheckpointsCmd(m.client, m.ctx, m.sessionID)
			case strings.HasPrefix(input, "/rewind "):
				if m.sessionID == "" {
					m.messages = append(m.messages, styleSystemMessage("⚠️ No active session"))
					m.updateViewport()
					return m, nil
				}
				// Parse /rewind <checkpoint_id> [--code] [--all]
				args := strings.TrimSpace(strings.TrimPrefix(input, "/rewind "))
				parts := strings.Fields(args)
				if len(parts) == 0 {
					m.messages = append(m.messages, styleSystemMessage("⚠️ Usage: /rewind <checkpoint_id> [--code] [--all]"))
					m.updateViewport()
					return m, nil
				}
				checkpointID := parts[0]
				rewindCode := false
				rewindConversation := true // default: rewind conversation
				for _, p := range parts[1:] {
					switch p {
					case "--code":
						rewindCode = true
						rewindConversation = false // Only code if --code alone
					case "--all":
						rewindCode = true
						rewindConversation = true
					}
				}
				m.textarea.Reset()
				m.waiting = true
				return m, rewindCmd(m.client, m.ctx, m.sessionID, checkpointID, rewindCode, rewindConversation)
			}

			// Add User Message to History
			m.messages = append(m.messages, styleUserMessage(input, m.renderer))
			m.textarea.Reset()
			m.waiting = true
			m.updateViewport()

			// If no session yet, create one first with the message
			if m.sessionID == "" {
				return m, createSessionWithMessageCmd(m.client, m.ctx, input)
			}

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
		m.sessionID = msg.sessionID
		m.waiting = false
		m.messages = append(m.messages, styleSystemMessage(fmt.Sprintf("Session Started (%s)", m.sessionID)))
		m.updateViewport()
		// Start Streaming Events interaction
		cmds := []tea.Cmd{streamEventsCmd(m.client, m.ctx, m.sessionID)}
		// If there's a pending message, send it now
		if msg.pendingMessage != "" {
			m.waiting = true
			cmds = append(cmds, sendMessageCmd(m.client, m.ctx, m.sessionID, msg.pendingMessage))
		}
		return m, tea.Batch(cmds...)

	case streamReadyMsg:
		m.eventCh = msg.ch
		return m, waitForNextEvent(m.eventCh)

	case permissionHandledMsg:
		// Clear permission request after successful submission
		m.permissionRequest = nil
		m.messages = append(m.messages, styleSystemMessage("✅ Permission submitted"))
		m.updateViewport()
		return m, nil

	case checkpointsListedMsg:
		m.waiting = false
		if len(msg.checkpoints) == 0 {
			m.messages = append(m.messages, styleSystemMessage("📋 No checkpoints found"))
		} else {
			cpMsg := styleSystemMsg.Render(fmt.Sprintf("📋 Checkpoints (%d total):\n", len(msg.checkpoints)))
			for i, cp := range msg.checkpoints {
				cpMsg += fmt.Sprintf("  %d. ID: %s | Messages: %d | Version: %d | Time: %s\n",
					i+1, cp.ID, cp.MessageCount, cp.StateVersion, cp.Timestamp.Format("2006-01-02 15:04:05"))
			}
			cpMsg += "\nUse '/rewind <checkpoint_id>' to restore (add --code or --all for code rewind)"
			m.messages = append(m.messages, cpMsg)
		}
		m.updateViewport()
		return m, nil

	case rewindCompletedMsg:
		m.waiting = false
		if msg.result.Success {
			successMsg := styleSystemMsg.Render(fmt.Sprintf("✅ %s\n", msg.result.Message))
			successMsg += fmt.Sprintf("  Restored to: %s (Version: %d, Messages: %d)",
				msg.result.RestoredCheckpoint.ID,
				msg.result.RestoredCheckpoint.StateVersion,
				msg.result.RestoredCheckpoint.MessageCount)
			m.messages = append(m.messages, successMsg)
		} else {
			m.messages = append(m.messages, styleSystemMessage(fmt.Sprintf("❌ %s", msg.result.Message)))
		}
		m.updateViewport()
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.textarea.SetWidth(msg.Width)
		// Account for status bar (1 line) + input area + spinner line
		statusBarHeight := 1
		if m.showStatusBar {
			statusBarHeight = 2
		}
		m.viewport.Height = msg.Height - m.textarea.Height() - statusBarHeight - 2
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
						// Parse arguments for card display
						var argsMap map[string]interface{}
						json.Unmarshal([]byte(tc.Arguments), &argsMap)
						m.messages = append(m.messages, RenderToolCall(tc.Name, argsMap, "running"))
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
				// Prevent UI lock: If we receive a result for the tool we are waiting on, clear the prompt
				if m.permissionRequest != nil && m.permissionRequest.ToolName == data.ToolName {
					m.permissionRequest = nil
				}

				if data.ToolName != "talk" {
					status := "success"
					if !data.Success {
						status = "error"
					}
					m.messages = append(m.messages, RenderToolCall(data.ToolName, nil, status))
				}
			}
		case "permission_request":
			var data struct {
				RequestID  string   `json:"request_id"`
				ToolName   string   `json:"tool_name"`
				Permission string   `json:"permission"`
				Patterns   []string `json:"patterns"`
			}
			if err := json.Unmarshal(msg.Data, &data); err == nil {
				// Set pending permission request (UI will render it in View)
				m.permissionRequest = &struct {
					RequestID      string
					ToolName       string
					Permission     string
					Patterns       []string
					SelectedOption int
				}{
					RequestID:      data.RequestID,
					ToolName:       data.ToolName,
					Permission:     data.Permission,
					Patterns:       data.Patterns,
					SelectedOption: 0, // Default to "Allow once"
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
	if m.permissionRequest != nil {
		s.WriteString(RenderPermissionRequest(
			m.permissionRequest.ToolName,
			m.permissionRequest.Permission,
			m.permissionRequest.Patterns,
			m.permissionRequest.SelectedOption,
		))
	} else if m.waiting {
		s.WriteString(m.spinner.View() + " Thinking...\n")
	} else {
		s.WriteString("\n")
	}

	// Input
	if m.permissionRequest == nil {
		s.WriteString(m.textarea.View())
	}

	// Status bar
	if m.showStatusBar && m.width > 0 {
		s.WriteString("\n")
		s.WriteString(RenderStatusBar(m.sessionID, m.model, m.width))
	}

	return s.String()
}

func (m *model) updateViewport() {
	m.viewport.SetContent(strings.Join(m.messages, "\n\n"))
	m.viewport.GotoBottom()
}

// Commands

func submitPermissionCmd(c *client.Client, ctx context.Context, sid, reqID string, approved, always bool) tea.Cmd {
	return func() tea.Msg {
		if err := c.SubmitPermission(ctx, sid, reqID, approved, always); err != nil {
			return errMsg(err)
		}
		return permissionHandledMsg{}
	}
}

// Commands

func createSessionCmd(c *client.Client, ctx context.Context) tea.Cmd {
	return createSessionWithMessageCmd(c, ctx, "")
}

func createSessionWithMessageCmd(c *client.Client, ctx context.Context, pendingMessage string) tea.Cmd {
	return func() tea.Msg {
		// Create session with empty prompt - no LLM call until message is sent
		sid, err := c.CreateSession(ctx, "")
		if err != nil {
			return errMsg(err)
		}
		return sessionCreatedMsg{sessionID: sid, pendingMessage: pendingMessage}
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

// Checkpoint messages
type checkpointsListedMsg struct {
	checkpoints []client.CheckpointResponse
}

type rewindCompletedMsg struct {
	result *client.RewindResponse
}

// Checkpoint commands
func listCheckpointsCmd(c *client.Client, ctx context.Context, sid string) tea.Cmd {
	return func() tea.Msg {
		resp, err := c.ListCheckpoints(ctx, sid)
		if err != nil {
			return errMsg(err)
		}
		return checkpointsListedMsg{checkpoints: resp.Checkpoints}
	}
}

func rewindCmd(c *client.Client, ctx context.Context, sid string, checkpointID string, rewindCode bool, rewindConversation bool) tea.Cmd {
	return func() tea.Msg {
		resp, err := c.Rewind(ctx, sid, checkpointID, rewindCode, rewindConversation)
		if err != nil {
			return errMsg(err)
		}
		return rewindCompletedMsg{result: resp}
	}
}

// Styles

func styleUserMessage(msg string, r *glamour.TermRenderer) string {
	return styleUserLabel.Render("❯ You") + "\n" + msg
}

func styleAssistantMessage(msg string) string {
	return styleAssistantLabel.Render("◆ Assistant") + "\n" + msg
}

func styleSystemMessage(msg string) string {
	return styleSystemMsg.Render(msg)
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
