package main

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/knz/bubbline"
	"github.com/knz/bubbline/editline"
)

// This would be your slow, external call (e.g., to an LLM)
func getSlowSuggestionFromServer(prompt string) string {
	// Simulate network latency
	time.Sleep(300 * time.Millisecond)

	history := []string{
		"SELECT * FROM users WHERE active = true",
		"SELECT * FROM products ORDER BY created_at DESC",
		"INSERT INTO users (name, email) VALUES ('John Doe', 'john.doe@example.com')",
	}

	lowerPrompt := strings.ToLower(prompt)
	if lowerPrompt == "" {
		return ""
	}

	for i := len(history) - 1; i >= 0; i-- {
		histEntry := history[i]
		if strings.HasPrefix(strings.ToLower(histEntry), lowerPrompt) && len(histEntry) > len(prompt) {
			return histEntry[len(prompt):]
		}
	}
	return ""
}

func main() {
	fmt.Println("Ghost text suggestion example (with async simulation).")
	fmt.Println("Start typing 'SELECT' or 'INSERT' (case-insensitive).")
	fmt.Println("Press the right arrow key to accept a suggestion.")
	fmt.Println()

	m := bubbline.New()

	// Set the new asynchronous suggestion provider
	m.SetSuggestionExec(func(buffer string) tea.Cmd {
		// This function returns a command.
		// The command is a function that returns a message.
		return func() tea.Msg {
			// This part runs in a background goroutine.
			suggestion := getSlowSuggestionFromServer(buffer)

			// Return the result wrapped in the message type
			// that the editline model now understands.
			return editline.SuggestionMsg(suggestion)
		}
	})

	suggestionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	m.FocusedStyle.Editor.Suggestion = suggestionStyle
	m.BlurredStyle.Editor.Suggestion = suggestionStyle

	// Main loop
	for {
		val, err := m.GetLine()

		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Println("\nBye!")
				break
			}
			if errors.Is(err, bubbline.ErrInterrupted) {
				fmt.Println("^C")
				continue
			}
			fmt.Println("error:", err)
			break
		}

		fmt.Printf("\nYou entered: %q\n", val)
		m.AddHistory(val)
	}
}
