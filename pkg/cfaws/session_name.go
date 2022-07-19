package cfaws

import "github.com/segmentio/ksuid"

// sessionName returns a unique session identifier for the aws console
// this ensures that user activity can be easily audited per session
// this uses the convenient ksuid library for generating unique IDs
func sessionName() string {
	// using the acronym gntd to ensure the id is not longer than 32 chars
	return "gntd-" + ksuid.New().String()
}
