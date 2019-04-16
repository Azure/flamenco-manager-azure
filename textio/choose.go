package textio

import (
	"context"
	"fmt"
)

// StrMap turns an array of strings into a map from those strings to true
func StrMap(choices []string) map[string]bool {
	themap := map[string]bool{}
	for _, choice := range choices {
		themap[choice] = true
	}
	return themap
}

// Choose presents a user with a set of valid choices, but also allows making another choice.
func Choose(ctx context.Context, choices []string, prompt string) (string, bool) {
	existing := StrMap(choices)
	prompt = fmt.Sprintf("%s %v", prompt, choices)
	choice := ReadLine(ctx, prompt)
	return choice, existing[choice]
}
