package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

func isBinary(filename string) bool {
	// Simple heuristic: check if the first 4 bytes are a common text encoding
	data, err := os.ReadFile(filename)
	if err != nil {
		return true // Error reading file, assume binary
	}
	return !isText(data)
}

func isText(data []byte) bool {
	for _, b := range data[:4] {
		if b < 0x20 || b > 0x7E {
			return false
		}
	}
	return true
}

func walkFiles(path string) (string, error) {
	var builder strings.Builder
	err := os.Chdir(path)
	if err != nil {
		log.Fatalf("Failed to change directory: %v", err)
	}
	cmd := exec.Command("git", "ls-files")
	output, err := cmd.Output()
	if err != nil {
		log.Fatalf("Failed to execute git ls-files: %v", err)
	}
	files := strings.Split(string(output), "\n")
	for _, filePath := range files {
		if filePath == "" {
			continue
		}
		_, err := os.Stat(filePath)
		if err != nil {
			return builder.String(), err
		}
		if isBinary(filePath) {
			fmt.Printf("skipping %s\n", filePath)
			continue // Skip binary files
		}
		fileContent, err := os.ReadFile(filePath)
		if err != nil {
			return builder.String(), err
		}
		fmt.Printf("reading %s\n", filePath)
		builder.WriteString(fmt.Sprintf("\nContent of %s:\n```\n%s\n```\n", filePath, string(fileContent)))
	}
	return builder.String(), nil
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	geminiApiKey := os.Getenv("GEMINI_API_KEY")
	folderPath := os.Getenv("FOLDER_PATH")
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(geminiApiKey))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-1.5-flash")

	content, err := walkFiles(folderPath)
	if err != nil {
		log.Fatal(err)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter prompt: ")
	prompt, _ := reader.ReadString('\n')
	prompt = fmt.Sprintf("Here are all the files in the current project:\n%s\n%s", content, prompt)

	fmt.Printf("prompt has %d characters\n", len(prompt))
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		log.Fatal(err)
	}

	if len(resp.Candidates) == 0 {
		log.Fatal("No candidates found in response")
	}
	for _, candidate := range resp.Candidates {
		fmt.Printf("%v", candidate.Content)
	}
}
