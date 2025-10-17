package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	choice         string
	result         string
	done           bool
	showingResult  bool
	quitting       bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.done {
			return m, tea.Quit
		}
		if m.showingResult {
			switch {
			case key.Matches(msg, keys.Confirm):
				m.result = performAction(m.choice, m.result)
				m.done = true
				return m, nil
			case key.Matches(msg, keys.Redo):
				m.result = generateMessage(m.choice)
				return m, nil
			case key.Matches(msg, keys.Back):
				m.choice = ""
				m.result = ""
				m.showingResult = false
				return m, nil
			}
		} else {
			switch {
			case key.Matches(msg, keys.Quit):
				m.quitting = true
				return m, tea.Quit
			case key.Matches(msg, keys.BigCommit):
				m.choice = "Big Commit"
				m.result = generateMessage(m.choice)
				m.showingResult = true
				return m, nil
			case key.Matches(msg, keys.ShortCommit):
				m.choice = "Short Concise Commit"
				m.result = generateMessage(m.choice)
				m.showingResult = true
				return m, nil
			case key.Matches(msg, keys.Stash):
				m.choice = "Stash with Message"
				m.result = generateMessage(m.choice)
				m.showingResult = true
				return m, nil
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return ""
	}
	if m.done {
		return m.result + "\n\nPress any key to exit"
	}
	if m.showingResult {
		return m.result + "\n\nc - Confirm | r - Redo | b - Back"
	}
	return mainView()
}

func mainView() string {
	style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	title := style.Render("Commiter - Choose an action:")
	
	instructions := `
b - Big Commit (detailed message)
s - Short Concise Commit (auto-commit)
t - Stash with Message
q - Quit
`
	
	return title + instructions
}



func generateMessage(choice string) string {
	switch choice {
	case "Big Commit":
		return generateCommitMessage(false)
	case "Short Concise Commit":
		return generateCommitMessage(true)
	case "Stash with Message":
		return generateStashMessage()
	default:
		return "Invalid choice"
	}
}

func generateCommitMessage(simple bool) string {
	apiKey := loadAPIKey()

	// Get git diff --staged
	cmd := exec.Command("git", "diff", "--staged")
	diff, err := cmd.Output()
	if err != nil {
		return fmt.Sprintf("Error getting git diff: %v", err)
	}
	if len(diff) == 0 {
		return "No staged changes"
	}

	// Construct prompt
	var prompt string
	if simple {
		prompt = loadSimplePrompt() + string(diff)
	} else {
		prompt = loadRegularPrompt() + string(diff)
	}

	// Call OpenRouter API
	reqBody := OpenRouterRequest{
		Model:    loadModel(),
		Messages: []Message{{Role: "user", Content: prompt}},
	}
	jsonData, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Sprintf("Error creating request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("Error making request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("Error reading response: %v", err)
	}
	if resp.StatusCode != 200 {
		return fmt.Sprintf("API error: %s", string(body))
	}

	var orResp OpenRouterResponse
	json.Unmarshal(body, &orResp)
	if len(orResp.Choices) == 0 {
		return "No response from API"
	}

	output := orResp.Choices[0].Message.Content

	return output
}

func generateStashMessage() string {
	apiKey := loadAPIKey()

	// Get git diff
	cmd := exec.Command("git", "diff")
	diff, err := cmd.Output()
	if err != nil {
		return fmt.Sprintf("Error getting git diff: %v", err)
	}
	if len(diff) == 0 {
		return "No changes to stash"
	}

	// Construct prompt for stash message
	prompt := loadSimplePrompt() + string(diff)

	// Call OpenRouter API
	reqBody := OpenRouterRequest{
		Model:    loadModel(),
		Messages: []Message{{Role: "user", Content: prompt}},
	}
	jsonData, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Sprintf("Error creating request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("Error making request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("Error reading response: %v", err)
	}
	if resp.StatusCode != 200 {
		return fmt.Sprintf("API error: %s", string(body))
	}

	var orResp OpenRouterResponse
	json.Unmarshal(body, &orResp)
	if len(orResp.Choices) == 0 {
		return "No response from API"
	}

	message := strings.TrimSpace(orResp.Choices[0].Message.Content)

	return message
}

func performAction(choice, message string) string {
	switch choice {
	case "Big Commit":
		return performCommit(message, false)
	case "Short Concise Commit":
		return performCommit(message, true)
	case "Stash with Message":
		return performStash(message)
	default:
		return "Invalid choice"
	}
}

func performCommit(message string, simple bool) string {
	// Copy to clipboard
	var copyCmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		copyCmd = exec.Command("pbcopy")
	} else {
		copyCmd = exec.Command("xclip", "-selection", "clipboard")
	}
	copyCmd.Stdin = strings.NewReader(message)
	err := copyCmd.Run()
	if err != nil {
		fmt.Printf("Error copying to clipboard: %v\n", err)
	}

	// Commit
	commitCmd := exec.Command("git", "commit", "-m", strings.TrimSpace(message))
	err = commitCmd.Run()
	if err != nil {
		return fmt.Sprintf("Error committing: %v", err)
	}
	return "Committed successfully."
}

func performStash(message string) string {
	// Stash with message
	stashCmd := exec.Command("git", "stash", "push", "-m", message)
	err := stashCmd.Run()
	if err != nil {
		return fmt.Sprintf("Error stashing: %v", err)
	}
	return fmt.Sprintf("Stashed with message: %s", message)
}

var keys = struct {
	Quit        key.Binding
	BigCommit   key.Binding
	ShortCommit key.Binding
	Stash       key.Binding
	Confirm     key.Binding
	Redo        key.Binding
	Back        key.Binding
}{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	BigCommit: key.NewBinding(
		key.WithKeys("b"),
		key.WithHelp("b", "big commit"),
	),
	ShortCommit: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "short commit"),
	),
	Stash: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "stash"),
	),
	Confirm: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "confirm"),
	),
	Redo: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "redo"),
	),
	Back: key.NewBinding(
		key.WithKeys("b"),
		key.WithHelp("b", "back"),
	),
}

func runTUI() error {
	m := model{}

	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}

type initModel struct {
	apiKey string
	done   bool
	err    error
}

func (m initModel) Init() tea.Cmd {
	return nil
}

func (m initModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if m.apiKey == "" {
				m.err = fmt.Errorf("API key cannot be empty")
				return m, nil
			}
			// Save the API key and defaults
			saveAPIKey(m.apiKey)
			saveSimplePrompt("Generate a short, concise commit message based on the provided Git differences below. Output only the commit message as a single line in lower case. Do not include any additional text, quotes, or explanations.\n\n---\nBEGIN GIT DIFF:\n")
			saveRegularPrompt("Generate a short, concise commit message based on the provided Git differences below.\nProvide up to 3 additional description options. Output in this exact format:\n\nfeat: commit message\n- desc option 1\n- desc option 2\n- optional desc option 3\n\nDo not include any other text.\n\n---\nBEGIN GIT DIFF:\n")
			saveModel("mistralai/ministral-3b")
			m.done = true
			return m, tea.Quit
		case tea.KeyBackspace, tea.KeyDelete:
			if len(m.apiKey) > 0 {
				m.apiKey = m.apiKey[:len(m.apiKey)-1]
			}
		case tea.KeyCtrlC:
			return m, tea.Quit
		default:
			if msg.Type == tea.KeyRunes {
				m.apiKey += string(msg.Runes)
			}
		}
	}
	return m, nil
}

func (m initModel) View() string {
	if m.done {
		return "✅ Setup complete! You can now use commiter.\nPress any key to exit."
	}

	var b strings.Builder

	// Title
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		Render("🚀 Welcome to Commiter")
	b.WriteString(title + "\n\n")

	// Instructions
	instructions := "Get started by visiting https://openrouter.ai/ and getting an API key.\n\n"
	b.WriteString(instructions)

	// Input prompt
	prompt := "Enter your OpenRouter API key: "
	if m.err != nil {
		prompt = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("❌ " + m.err.Error() + "\n") + prompt
	}
	b.WriteString(prompt)

	// Show current input (masked for security)
	masked := strings.Repeat("*", len(m.apiKey))
	b.WriteString(masked)

	// Cursor
	b.WriteString("█")

	b.WriteString("\n\nPress Enter to submit, Ctrl+C to quit")

	return b.String()
}

func runInit() error {
	m := initModel{}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	// If not done, user quit
	if !finalModel.(initModel).done {
		os.Exit(0)
	}

	return nil
}

// Existing types and functions moved here

type OpenRouterRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenRouterResponse struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Message Message `json:"message"`
}

func getConfigDir() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		fmt.Println("Error getting config dir:", err)
		os.Exit(1)
	}
	return filepath.Join(configDir, "commiter")
}

func loadAPIKey() string {
	configDir := getConfigDir()
	keyFile := filepath.Join(configDir, "api_key")
	data, err := os.ReadFile(keyFile)
	if err != nil {
		fmt.Println("API key not found. Run 'commiter init' to set it up.")
		os.Exit(1)
	}
	return strings.TrimSpace(string(data))
}

func saveAPIKey(key string) {
	configDir := getConfigDir()
	os.MkdirAll(configDir, 0755)
	keyFile := filepath.Join(configDir, "api_key")
	err := os.WriteFile(keyFile, []byte(key), 0600)
	if err != nil {
		fmt.Println("Error saving API key:", err)
		os.Exit(1)
	}
	fmt.Println("API key saved successfully.")
}

func loadSimplePrompt() string {
	configDir := getConfigDir()
	promptFile := filepath.Join(configDir, "simple_prompt")
	data, err := os.ReadFile(promptFile)
	if err != nil {
		// Return default prompt if file doesn't exist
		return "Generate a short, concise commit message based on the provided Git differences below. Output only the commit message as a single line in lower case. Do not include any additional text, quotes, or explanations.\n\n---\nBEGIN GIT DIFF:\n"
	}
	return strings.TrimSpace(string(data))
}

func loadRegularPrompt() string {
	configDir := getConfigDir()
	promptFile := filepath.Join(configDir, "regular_prompt")
	data, err := os.ReadFile(promptFile)
	if err != nil {
		// Return default prompt if file doesn't exist
		return "Generate a short, concise commit message based on the provided Git differences below.\nProvide up to 3 additional description options. Output in this exact format:\n\nfeat: commit message\n- desc option 1\n- desc option 2\n- optional desc option 3\n\nDo not include any other text.\n\n---\nBEGIN GIT DIFF:\n"
	}
	return strings.TrimSpace(string(data))
}

func saveSimplePrompt(prompt string) {
	configDir := getConfigDir()
	os.MkdirAll(configDir, 0755)
	promptFile := filepath.Join(configDir, "simple_prompt")
	err := os.WriteFile(promptFile, []byte(prompt), 0644)
	if err != nil {
		fmt.Println("Error saving simple prompt:", err)
		os.Exit(1)
	}
}

func saveRegularPrompt(prompt string) {
	configDir := getConfigDir()
	os.MkdirAll(configDir, 0755)
	promptFile := filepath.Join(configDir, "regular_prompt")
	err := os.WriteFile(promptFile, []byte(prompt), 0644)
	if err != nil {
		fmt.Println("Error saving regular prompt:", err)
		os.Exit(1)
	}
}

func loadModel() string {
	configDir := getConfigDir()
	modelFile := filepath.Join(configDir, "model")
	data, err := os.ReadFile(modelFile)
	if err != nil {
		// Return default model if file doesn't exist
		return "mistralai/ministral-3b"
	}
	return strings.TrimSpace(string(data))
}

func saveModel(model string) {
	configDir := getConfigDir()
	os.MkdirAll(configDir, 0755)
	modelFile := filepath.Join(configDir, "model")
	err := os.WriteFile(modelFile, []byte(model), 0644)
	if err != nil {
		fmt.Println("Error saving model:", err)
		os.Exit(1)
	}
}