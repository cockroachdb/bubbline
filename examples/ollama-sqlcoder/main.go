// examples/ollama-sqlcoder/main.go
// you need to ollama run sqlcoder before running this
// this looks for ollama on port 11434
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/knz/bubbline"
	"github.com/knz/bubbline/editline"
)

// --- Ollama API Communication ---

const ollamaURL = "http://localhost:11434/api/generate"
const modelName = "sqlcoder" // The name of the model you have downloaded in Ollama

// Create a custom HTTP client with a longer timeout.
var ollamaClient = &http.Client{
	Timeout: 60 * time.Second,
}

// OllamaRequest represents the JSON payload to send to the Ollama API.
type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	// Adding System prompt to guide the model's behavior.
	System string `json:"system"`
}

// OllamaResponse represents a single JSON object in the response from Ollama.
type OllamaResponse struct {
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	Response  string    `json:"response"`
	Done      bool      `json:"done"`
}

// getLLMSuggestion makes a network request to the local Ollama server.
func getLLMSuggestion(prompt string) (string, error) {
	// 1. Define a system prompt to constrain the model.
	// This tells sqlcoder to act only as a SQL completer.
	systemPrompt := "You are an expert SQL assistant. Complete the following SQL code. Do not add any explanations or conversational text. Only provide the SQL code completion."

	// 2. Construct the request payload with the new system prompt.
	requestData, err := json.Marshal(OllamaRequest{
		Model:  modelName,
		System: systemPrompt,
		Prompt: prompt,
		Stream: false,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// 3. Make the HTTP POST request using our custom client.
	resp, err := ollamaClient.Post(ollamaURL, "application/json", bytes.NewBuffer(requestData))
	if err != nil {
		return "", fmt.Errorf("failed to connect to ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama returned non-200 status: %s", resp.Status)
	}

	// 4. Decode the JSON response.
	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("failed to decode ollama response: %w", err)
	}

	// 5. Clean up the response to get a good suggestion.
	suggestion := strings.TrimPrefix(ollamaResp.Response, prompt)
	suggestion = strings.TrimLeft(suggestion, "\n\r \t")
	if firstLine, _, ok := strings.Cut(suggestion, "\n"); ok {
		suggestion = firstLine
	}

	return suggestion, nil
}

// --- Debouncing state ---
var (
	lastRequestTime time.Time
	mu              sync.Mutex
)

const debounceDuration = 400 * time.Millisecond

func main() {
	fmt.Printf("SQLCoder AI Assistant (Model: %s)\n", modelName)
	fmt.Println("Start typing a SQL query. Suggestions will appear after you pause.")
	fmt.Println()

	m := bubbline.New()
	m.Prompt = "cockroach> "

	// Configure the asynchronous suggestion provider with debouncing.
	m.SetSuggestionExec(func(buffer string) tea.Cmd {
		requestTime := time.Now()
		mu.Lock()
		lastRequestTime = requestTime
		mu.Unlock()

		return func() tea.Msg {
			time.Sleep(debounceDuration)

			mu.Lock()
			isStale := lastRequestTime.After(requestTime)
			mu.Unlock()

			if isStale {
				return nil
			}

			suggestion, err := getLLMSuggestion(buffer)
			if err != nil {
				log.Printf("Error getting suggestion: %v", err)
				return editline.SuggestionMsg("")
			}

			return editline.SuggestionMsg(suggestion)
		}
	})

	suggestionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	m.FocusedStyle.Editor.Suggestion = suggestionStyle
	m.BlurredStyle.Editor.Suggestion = suggestionStyle

	// --- Main editor loop ---
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
