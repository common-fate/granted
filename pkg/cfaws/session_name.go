package cfaws

import "github.com/segmentio/ksuid"

// sessionName returns a unique session identifier for the aws console
// this ensures that user activity can be easily audited per session
func sessionName() string {
	return "gran-" + ksuid.New().String()
}
