package command

import "github.com/BKrajancic/FLD-Bot/m/v2/src/service"

// Return the received message
func Repeater(sender service.Conversation, user service.User, msg [][]string, sink func(service.Conversation, service.Message)) {
	sink(sender, service.Message{Description: msg[0][1]})
}
