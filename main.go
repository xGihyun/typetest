package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const DURATION = time.Second * 10

func main() {
	words, err := getWords(200)
	if err != nil {
		log.Fatal(err)
	}
	text := strings.Join(words, " ")

	p := tea.NewProgram(initialModel(text))
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

type model struct {
	textInput      textinput.Model
	help           help.Model
	keymap         keymap
	ghostText      string
	wordsPerMinute int
	timer          timer.Model
	started        bool
	incorrectCount int
	accuracy       float32
	errorPositions map[int]bool
	maxTyped       int
}

type keymap struct{}

func (k keymap) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "quit")),
	}
}

func (k keymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}

func initialModel(text string) model {
	ti := textinput.New()
	ti.Width = 80
	ti.Focus()

	return model{
		textInput:      ti,
		help:           help.New(),
		keymap:         keymap{},
		ghostText:      text,
		timer:          timer.New(DURATION),
		wordsPerMinute: 0,
		started:        false,
		accuracy:       100,
		errorPositions: make(map[int]bool),
		maxTyped:       0,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var initTimerCmd tea.Cmd
	cursorPos := m.textInput.Position()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyBackspace:
			if cursorPos >= len(m.ghostText) {
				break
			}

			if m.ghostText[cursorPos] == ' ' && m.ghostText[cursorPos-1] == ' ' {
				m.ghostText = m.ghostText[:cursorPos-1] + m.ghostText[cursorPos:]
			}

		case tea.KeySpace:
			if m.timer.Timedout() {
				return m, nil
			}

			nextWordPos := strings.Index(m.ghostText[cursorPos:], " ")
			cur := m.textInput.Value()
			newStr := cur + strings.Repeat(" ", nextWordPos)
			m.textInput.SetValue(newStr)
			m.textInput.SetCursor(len(newStr))

		case tea.KeyRunes:
			if m.timer.Timedout() {
				return m, nil
			}

			if m.ghostText[cursorPos] == ' ' {
				m.ghostText = m.ghostText[:cursorPos] + " " + m.ghostText[cursorPos:]
			}

			if !m.started {
				m.started = true
				initTimerCmd = m.timer.Init()
			}
		}

		m.textInput, cmd = m.textInput.Update(msg)

		if m.started {
			m.calculateAccuracy()
		}

		return m, tea.Batch(cmd, initTimerCmd)

	case timer.TickMsg:
		if m.timer.Timedout() {
			return m, nil
		}

		v := m.textInput.Value()
		words := float64(len(v)) / 5.0
		elapsed := DURATION.Seconds() - m.timer.Timeout.Seconds()
		if elapsed >= 0.1 {
			wpm := words * (60 / elapsed)
			m.wordsPerMinute = int(wpm)
		}

		m.timer, cmd = m.timer.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *model) calculateAccuracy() {
	input := m.textInput.Value()
	if len(input) <= 0 {
		return
	}

	for i := 0; i < len(input); i++ {
		typedChar := input[i]
		ghostChar := m.ghostText[i]
		if typedChar != ghostChar {
			m.errorPositions[i] = true
		}
	}

	if len(input) > m.maxTyped {
		m.maxTyped = len(input)
		correct := len(input) - len(m.errorPositions)
		m.accuracy = (float32(correct) / float32(len(input))) * 100
	}
}

var (
	ghostTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))
	correctTextStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("2"))
	incorrectTextStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("1"))
	cursorStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("7")).
			Foreground(lipgloss.Color("0"))
)

func (m model) View() string {
	var builder strings.Builder

	// NOTE: There might be an issue with using `[]rune` here since we only use `string` on `model.Update`
	// But we don't use special characters so it would be fine for now.
	ghostRunes := []rune(m.ghostText)
	typedRunes := []rune(m.textInput.Value())
	cursorPos := m.textInput.Position()
	currentLineLength := 0

	for i, ghostChar := range ghostRunes {
		if currentLineLength >= m.textInput.Width && ghostChar == ' ' {
			builder.WriteByte('\n')
			currentLineLength = 0
			continue
		}

		if i < len(typedRunes) {
			typedChar := typedRunes[i]
			if typedChar == ghostChar {
				builder.WriteString(correctTextStyle.Render(string(ghostChar)))
			} else if typedChar == ' ' && ghostChar != ' ' {
				builder.WriteString(ghostTextStyle.Render(string(ghostChar)))
			} else {
				builder.WriteString(incorrectTextStyle.Render(string(typedChar)))
			}
		} else {
			if i == cursorPos {
				builder.WriteString(cursorStyle.Render(string(ghostChar)))
			} else {
				builder.WriteString(ghostTextStyle.Render(string(ghostChar)))
			}
		}

		if ghostChar == '\n' {
			currentLineLength = 0
		} else {
			currentLineLength++
		}
	}

	if m.timer.Timedout() {
		m.textInput.Blur()
		builder.Reset()
		builder.WriteString(fmt.Sprintf("WPM: %d\nACC: %.2f%%", m.wordsPerMinute, m.accuracy))
	}

	return fmt.Sprintf(
		"Type the stuff:\n\nTIME: %s\nWPM:  %d\nACC:  %.2f%%\n\n%s\n\n%s",
		m.timer.View(),
		m.wordsPerMinute,
		m.accuracy,
		builder.String(),
		m.help.View(m.keymap),
	)
}
