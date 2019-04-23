package azssh

import (
	"bufio"
	"io"
)

// LineReader scans the reader line-by-line and sends those lines to the channel.
// The returned function can be used to obtain the first non-EOF error seen by the scanner.
func LineReader(reader io.Reader) (<-chan string, func() error) {
	channel := make(chan string)
	var err error

	go func() {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			channel <- scanner.Text()
		}

		err = scanner.Err()
	}()

	return channel, func() error {
		return err
	}
}
