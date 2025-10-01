package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var apiKey string

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
		fmt.Println("API key not found. Run 'commiter --init' to set it up.")
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

func main() {
	initFlag := flag.Bool("init", false, "Initialize with OpenRouter API key")
	simple := flag.Bool("simple", false, "Generate simple one-liner commit message")
	flag.Parse()

	if *initFlag {
		fmt.Print("Enter your OpenRouter API key: ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		key := scanner.Text()
		if key == "" {
			fmt.Println("API key cannot be empty.")
			os.Exit(1)
		}
		saveAPIKey(key)
		return
	}

	apiKey = loadAPIKey()

	// Get git diff --staged
	cmd := exec.Command("git", "diff", "--staged")
	diff, err := cmd.Output()
	if err != nil {
		fmt.Println("Error getting git diff:", err)
		os.Exit(1)
	}
	if len(diff) == 0 {
		fmt.Println("No staged changes")
		os.Exit(1)
	}

	// Construct prompt
	var prompt string
	if *simple {
		prompt = "Generate a short, concise commit message based on the provided Git differences below. Output only the commit message as a single line in lower case. Do not include any additional text, quotes, or explanations.\n\n---\nBEGIN GIT DIFF:\n" + string(diff)
	} else {
		prompt = "Generate a short, concise commit message based on the provided Git differences below.\nProvide up to 3 additional description options. Output in this exact format:\n\nfeat: commit message\n- desc option 1\n- desc option 2\n- optional desc option 3\n\nDo not include any other text.\n\n---\nBEGIN GIT DIFF:\n" + string(diff)
	}

	// Call OpenRouter API
	reqBody := OpenRouterRequest{
		Model:    "mistralai/ministral-3b",
		Messages: []Message{{Role: "user", Content: prompt}},
	}
	jsonData, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error creating request:", err)
		os.Exit(1)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response:", err)
		os.Exit(1)
	}
	if resp.StatusCode != 200 {
		fmt.Println("API error:", string(body))
		os.Exit(1)
	}

	var orResp OpenRouterResponse
	json.Unmarshal(body, &orResp)
	if len(orResp.Choices) == 0 {
		fmt.Println("No response from API")
		os.Exit(1)
	}

	output := orResp.Choices[0].Message.Content

	// Output to stdout
	fmt.Print(output)

	// Copy to clipboard
	var copyCmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		copyCmd = exec.Command("pbcopy")
	} else {
		copyCmd = exec.Command("xclip", "-selection", "clipboard")
	}
	copyCmd.Stdin = strings.NewReader(output)
	err = copyCmd.Run()
	if err != nil {
		fmt.Println("Error copying to clipboard:", err)
	}
}
