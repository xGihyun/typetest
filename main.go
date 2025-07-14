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

const DURATION = time.Second * 30

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

	h := help.New()
	km := keymap{}

	return model{
		textInput:      ti,
		help:           h,
		keymap:         km,
		ghostText:      text,
		timer:          timer.New(DURATION),
		wordsPerMinute: 0,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.timer.Init())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
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
			nextWordPos := strings.Index(m.ghostText[cursorPos:], " ")
			cur := m.textInput.Value()
			newStr := cur + strings.Repeat(" ", nextWordPos)
			m.textInput.SetValue(newStr)
			m.textInput.SetCursor(len(newStr))

		case tea.KeyRunes:
			if m.ghostText[cursorPos] != ' ' {
				break
			}

			m.ghostText = m.ghostText[:cursorPos] + " " + m.ghostText[cursorPos:]
		}

	case timer.TickMsg:
		v := m.textInput.Value()
		words := float64(len(v)) / 5.0
		elapsed := DURATION.Seconds() - m.timer.Timeout.Seconds()
		wpm := words * (60 / elapsed)

		if wpm > 0 {
			m.wordsPerMinute = int(wpm)
		}

		m.timer, cmd = m.timer.Update(msg)
		return m, cmd
	}

	m.textInput, cmd = m.textInput.Update(msg)

	return m, cmd
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

	return fmt.Sprintf(
		"Type the stuff:\n\n%s\n%d\n\n%s\n\n%s",
		m.timer.View(),
		m.wordsPerMinute,
		builder.String(),
		m.help.View(m.keymap),
	)
}
