package cfaws

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// sessionName returns a unique session identifier for the aws console
// this ensures that user activity can be easily audited per session
func TestSessionName(t *testing.T) {
	//getfederationtoken fails if name is longer than 32 characters long
	name := sessionName()
	assert.LessOrEqual(t, len(name), 32)
}
