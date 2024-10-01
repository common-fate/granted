// Package chromemsg implements the Chrome native messaging protocol:
// https://developer.chrome.com/docs/extensions/develop/concepts/native-messaging
package chromemsg

import (
	"io"
	"testing"
)

// TestServer_ReadWrite tests the protocol by creating a server where the output
// is connected to the input. We write to the output, and then try and read the
// same message back from the input.
func TestServer_ReadWrite(t *testing.T) {
	r, w := io.Pipe()

	server := &Server{Input: r, Output: w}

	message := []byte("Hello, Chrome!")

	go func() {
		if _, err := server.Write(message); err != nil {
			t.Errorf("failed to write message: %v", err)
		}
	}()

	readBuffer := make([]byte, len(message))
	if _, err := server.Read(readBuffer); err != nil {
		t.Errorf("failed to read message: %v", err)
	}

	if string(readBuffer) != string(message) {
		t.Errorf("expected %s, got %s", message, readBuffer)
	}
}
