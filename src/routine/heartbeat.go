package routine

import (
	"time"

	"github.com/BKrajancic/FLD-Bot/m/v2/src/service"
)

// Sends a message every now and again using parameter route.
// This is only useful for testing purposes.
func Heartbeat(delay time.Duration, destination service.Conversation, msg service.Message, route func(service.Conversation, service.Message)) {
	for _ = range time.Tick(delay) {
		route(destination, msg)
	}
}
