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
	Event  eventType
	Params []string
}

func handleEvent(dazeus *DaZeus, message Message) error {
	evt, err := makeEvent(message)

	if err != nil {
		return err
	}

	replier := makeReplier(dazeus, evt.Params[0], evt.Params[2], evt.Params[1])

	for _, l := range dazeus.listeners {
		if l.event == evt.Event && (l.event != EventCommand || l.command == evt.Params[3]) {
			dazeus.logger.Print("Calling matching event handler")
			l.handler(evt, replier)
		}
	}

	return nil
}

func makeEvent(message Message) (Event, error) {
	var event Event
	messageEventType, ok := message["event"].(string)

	if !ok {
		return event, errors.New("Could not find event type in message")
	}

	params, err := makeStringArray(message["params"])

	if err != nil {
		return event, err
	}

	evtType := eventType(messageEventType)
	event = Event{evtType, params}
	return event, nil
}
