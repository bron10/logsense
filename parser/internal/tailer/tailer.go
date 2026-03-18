package tailer

import (
	"bufio"
	"io"
	"log"
	"os"
	"time"
)

// LineCh is a channel that receives lines read from a tailed file.
type LineCh = chan string

// Tail opens a file and polls for new lines, sending each to the returned channel.
// It starts reading from the beginning of the file and continues polling at pollInterval.
// The channel is closed when done is closed.
func Tail(path string, pollInterval time.Duration, done <-chan struct{}) LineCh {
	ch := make(chan string, 100)

	go func() {
		defer close(ch)

		f, err := os.Open(path)
		if err != nil {
			log.Printf("tailer: failed to open %s: %v", path, err)
			return
		}
		defer f.Close()

		reader := bufio.NewReader(f)

		for {
			line, err := reader.ReadString('\n')
			if len(line) > 0 {
				// Trim trailing newline
				if line[len(line)-1] == '\n' {
					line = line[:len(line)-1]
				}
				if len(line) > 0 && line[len(line)-1] == '\r' {
					line = line[:len(line)-1]
				}
				select {
				case ch <- line:
				case <-done:
					return
				}
				continue
			}

			if err == io.EOF {
				// Poll for new data
				select {
				case <-time.After(pollInterval):
					continue
				case <-done:
					return
				}
			} else if err != nil {
				log.Printf("tailer: read error on %s: %v", path, err)
				return
			}
		}
	}()

	return ch
}
