package textio

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
)

var mutex = sync.Mutex{}

// ReadLine reads a line from stdin and returns it as string.
func ReadLine(ctx context.Context, prompt string) string {
	mutex.Lock()
	defer mutex.Unlock()

	fmt.Printf("%s: ", prompt)

	textChan := make(chan string)
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		textChan <- scanner.Text()
	}()

	select {
	case <-ctx.Done():
		fmt.Println("aborted")
		return ""
	case text := <-textChan:
		return strings.TrimSpace(text)
	}
}
