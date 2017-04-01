package dazeus

import (
	"errors"
)

type eventType string

const (
	// EventConnect is a connect event
	EventConnect eventType = "CONNECT"
	// EventDisconnect is a disconnect event
	EventDisconnect eventType = "DISCONNECT"
	// EventJoin is a join event
	EventJoin eventType = "JOIN"
	// EventPart is a part event
	EventPart eventType = "PART"
	// EventQuit is a quit event
	EventQuit eventType = "QUIT"
	// EventNick is a nick event
	EventNick eventType = "NICK"
	// EventMode is a mode event
	EventMode eventType = "MODE"
	// EventTopic is a topic event
	EventTopic eventType = "TOPIC"
	// EventInvite is an invite event
	EventInvite eventType = "INVITE"
	// EventKick is a kick event
	EventKick eventType = "KICK"
	// EventPrivMsg is a privmsg event
	EventPrivMsg eventType = "PRIVMSG"
	// EventNotice is a notice event
	EventNotice eventType = "NOTICE"
	// EventCtcp is a ctcp event
	EventCtcp eventType = "CTCP"
	// EventCtcpReply is a CTCP reply event
	EventCtcpReply eventType = "CTCP_REP"
	// EventAction is an action event
	EventAction eventType = "ACTION"
	// EventNumeric is a numeric event
	EventNumeric eventType = "NUMERIC"
	// EventUnknown is an unknown event
	EventUnknown eventType = "UNKNOWN"
	// EventWhois is a whois event
	EventWhois eventType = "WHOIS"
	// EventNames is a names event
	EventNames eventType = "NAMES"
	// EventPrivMsgMe is a privmsg from the bot itself
	EventPrivMsgMe eventType = "PRIVMSG_ME"
	// EventCtcpMe is CTCP message from the bot itself
	EventCtcpMe eventType = "CTCP_ME"
	// EventActionMe is an action event from the bot itself
	EventActionMe eventType = "ACTION_ME"
	// EventPong is a pong event
	EventPong eventType = "PONG"
	// EventCommand indicates any command event
	EventCommand eventType = "COMMAND"
)

// Event represents an event message
type Event struct {
	Event   eventType
	Params  []string
	DaZeus  *DaZeus
	Network string
	Channel string
	Sender  string
	Command string
}

// Reply allows an event handler to respond to the event with a message
func (event *Event) Reply(message string, highlight bool) error {
	return event.DaZeus.Reply(event.Network, event.Channel, event.Sender, message, highlight)
}

// ReplyAction allows an event handler to respond to the event with a ctcp action
func (event *Event) ReplyAction(message string) error {
	return event.DaZeus.ReplyAction(event.Network, event.Channel, event.Sender, message)
}

// ReplyNotice allows an event handler to respond to the event with a notice
func (event *Event) ReplyNotice(message string, highlight bool) error {
	return event.DaZeus.ReplyNotice(event.Network, event.Channel, event.Sender, message, highlight)
}

// ReplyCtcpReply allows an event handler to respond to the event with a ctcp reply
func (event *Event) ReplyCtcpReply(message string) error {
	return event.DaZeus.ReplyCtcpReply(event.Network, event.Channel, event.Sender, message)
}

func handleEvent(dazeus *DaZeus, message Message) error {
	evt, err := makeEvent(dazeus, message)

	if err != nil {
		return err
	}

	for _, l := range dazeus.listeners {
		if l.event == evt.Event && (l.event != EventCommand || l.command == evt.Command) {
			dazeus.logger.Print("Calling matching event handler")
			l.handler(evt)
		}
	}

	return nil
}

func makeEvent(dazeus *DaZeus, message Message) (Event, error) {
	var event Event
	messageEventType, ok := message["event"].(string)

	if !ok {
		return event, errors.New("Could not find event type in message")
	}

	params, err := makeStringArray(message["params"])

	if err != nil {
		return event, err
	}

	var network, channel, sender string
	network = params[0]

	if len(params) < 2 {
		sender = ""
		channel = ""
		params = params[1:]
	} else if len(params) < 3 {
		sender = params[1]
		channel = ""
		params = params[2:]
	} else {
		sender = params[1]
		channel = params[2]
		params = params[3:]
	}

	command := ""
	if messageEventType == "COMMAND" {
		command = params[0]
		params = params[1:]
	}

	evtType := eventType(messageEventType)
	event = Event{
		Event:   evtType,
		Params:  params,
		DaZeus:  dazeus,
		Network: network,
		Channel: channel,
		Sender:  sender,
		Command: command,
	}

	return event, nil
}
