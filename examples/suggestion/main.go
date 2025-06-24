package main

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/knz/bubbline"
)

func main() {
	fmt.Println("Ghost text suggestion example.")
	fmt.Println("Start typing 'SELECT' or 'INSERT' (case-insensitive).")
	fmt.Println("Press the right arrow key to accept a suggestion.")
	fmt.Println()

	m := bubbline.New()

	prepopulatedHistory := []string{
		"SELECT * FROM users WHERE active = true",
		"SELECT * FROM products ORDER BY created_at DESC",
		"INSERT INTO users (name, email) VALUES ('John Doe', 'john.doe@example.com')",
	}
	m.SetHistory(prepopulatedHistory)

	suggestionExec := func(currentBuffer string) string {
		if currentBuffer == "" {
			return ""
		}
		lowerCurrentBuffer := strings.ToLower(currentBuffer)
		history := m.GetHistory()
		for i := len(history) - 1; i >= 0; i-- {
			histEntry := history[i]
			if strings.HasPrefix(strings.ToLower(histEntry), lowerCurrentBuffer) && len(histEntry) > len(currentBuffer) {
				return histEntry[len(currentBuffer):]
			}
		}
		return ""
	}

	m.SetSuggestionExec(suggestionExec)

	suggestionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	m.FocusedStyle.Editor.Suggestion = suggestionStyle
	m.BlurredStyle.Editor.Suggestion = suggestionStyle

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
