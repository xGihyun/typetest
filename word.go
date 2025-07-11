package main

import (
	"math/rand"
	"os"
	"strings"
)

func getWords(count int) ([]string, error) {
	contents, err := os.ReadFile("words.txt")
	if err != nil {
		return nil, err
	}

	words := strings.Split(string(contents), "\n")
	var selectedWords []string

	for range count {
		randIdx := rand.Intn(len(words) - 1)
		randWord := words[randIdx]
		selectedWords = append(selectedWords, randWord)
	}

	return selectedWords, nil
}
