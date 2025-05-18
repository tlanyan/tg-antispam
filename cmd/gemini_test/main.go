package main

import (
	"fmt"
	"os"

	"tg-antispam/internal/handler"
)

func main() {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		panic("GEMINI_API_KEY not set")
	}

	if len(os.Args) < 2 {
		panic("Usage: ./gemini_test <message>")
	}

	message := os.Args[1]

	got, err := handler.ClassifyWithGemini(apiKey, "gemini-2.0-flash", message)
	if err != nil {
		panic(err)
	}
	fmt.Println(got)
}
