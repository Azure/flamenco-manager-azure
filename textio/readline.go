/* (c) 2019, Blender Foundation
 *
 * Permission is hereby granted, free of charge, to any person obtaining
 * a copy of this software and associated documentation files (the
 * "Software"), to deal in the Software without restriction, including
 * without limitation the rights to use, copy, modify, merge, publish,
 * distribute, sublicense, and/or sell copies of the Software, and to
 * permit persons to whom the Software is furnished to do so, subject to
 * the following conditions:
 *
 * The above copyright notice and this permission notice shall be
 * included in all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
 * EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
 * MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
 * IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
 * CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
 * TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
 * SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 */

package textio

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

var mutex = sync.Mutex{}

// ReadLine reads a line from stdin and returns it as string.
func ReadLine(ctx context.Context, prompt string) string {
	line, _ := readline(ctx, prompt)
	return line
}

// ReadLineWithDefault acts as ReadLine() but returns a default value when the user presses enter.
func ReadLineWithDefault(ctx context.Context, prompt, defaultValue string) string {
	if defaultValue == "" {
		return ReadLine(ctx, prompt)
	}

	line, ok := readline(ctx, fmt.Sprintf("%s [%s]", prompt, defaultValue))
	if !ok {
		return ""
	}
	if line == "" {
		return defaultValue
	}
	return line
}

func readline(ctx context.Context, prompt string) (string, bool) {
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
		return "", false
	case text := <-textChan:
		return strings.TrimSpace(text), true
	}
}

// ReadNonNegativeInt reads a line from stdin and returns it as int.
func ReadNonNegativeInt(ctx context.Context, prompt string, defaultZero bool) int {
	line := ReadLine(ctx, prompt)

	if line == "" {
		if defaultZero {
			return 0
		}
		logrus.Fatal("no input given, aborting")
	}

	asInt, err := strconv.Atoi(line)
	if err != nil {
		logrus.WithError(err).Fatal("invalid integer")
	}
	if asInt < 0 {
		logrus.WithField("input", asInt).Fatal("number must be non-negative integer")
	}

	return asInt
}
