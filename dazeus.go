package dazeus

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
)

// Message is a message as send by or received from the core.
type Message map[string]interface{}

// listener stores a listener internally in the plugin
type listener struct {
	event   eventType
	command string
	handler Handler
}

// Handler defines the function type for registering a callback
type Handler func(Event)

// ListenerHandle is used to register and unregister event callbacks
type ListenerHandle int

// DaZeus contains the connection information for a connection to the dazeus core
type DaZeus struct {
	conn          net.Conn
	buffer        bytes.Buffer
	listeners     map[ListenerHandle]listener
	lastHandle    ListenerHandle
	logger        *log.Logger
	callDepth     int
	responseQueue []Message
}

// Connect creates a new connection to a DaZeus core with logging to a Discard logger
func Connect(connectionString string) (*DaZeus, error) {
	logger := log.New(ioutil.Discard, "[dazeus-go] ", 0)
	return ConnectWithLogger(connectionString, logger)
}

// ConnectWithLoggingToStdErr creates a new connection and sets up basic logging to stderr
func ConnectWithLoggingToStdErr(connectionString string) (*DaZeus, error) {
	logger := log.New(os.Stderr, "[dazeus-go] ", log.LstdFlags)
	return ConnectWithLogger(connectionString, logger)
}

// ConnectWithLogger creates a new connection to a DaZeus core with the specified logging instance
func ConnectWithLogger(connectionString string, logger *log.Logger) (*DaZeus, error) {
	parts := strings.SplitN(connectionString, ":", 2)
	if len(parts) != 2 {
		return nil, errors.New("Invalid connection string")
	}

	format := parts[0]
	address := parts[1]

	if format != "tcp" && format != "unix" {
		return nil, errors.New("No such connection format")
	}

	conn, err := net.Dial(format, address)

	if err != nil {
		return nil, err
	}

	return &DaZeus{
		conn:          conn,
		buffer:        bytes.Buffer{},
		listeners:     make(map[ListenerHandle]listener, 0),
		lastHandle:    1,
		logger:        logger,
		callDepth:     0,
		responseQueue: make([]Message, 0),
	}, nil
}

// Listen starts listening for incoming events, this call is blockin
func (dazeus *DaZeus) Listen() error {
	for {
		err := waitForEvent(dazeus)
		if err != nil {
			return err
		}
	}
}

// Close closes the connection
func (dazeus *DaZeus) Close() error {
	dazeus.buffer.Reset()
	return dazeus.conn.Close()
}

// Subscribe registers a handle to receive events
func (dazeus *DaZeus) Subscribe(event eventType, handler Handler) (ListenerHandle, error) {
	ldata := listener{event, "", handler}

	dazeus.logger.Printf("Requesting core subscription for events of type '%s'", event)
	_, err := writeForSuccessResponse(dazeus, map[string]interface{}{
		"do":     "subscribe",
		"params": []string{string(event)},
	})

	if err != nil {
		return -1, err
	}

	handle := dazeus.lastHandle
	dazeus.lastHandle++
	dazeus.listeners[handle] = ldata

	return handle, nil
}

// SubscribeCommand allows the user to subscribe to a command
func (dazeus *DaZeus) SubscribeCommand(command string, scope Scope, handler Handler) (ListenerHandle, error) {
	ldata := listener{EventCommand, command, handler}

	scopeSlice, err := scope.ToCommandSlice()
	if err != nil {
		return -1, err
	}

	dazeus.logger.Printf("Requesting core subscription for command '%s'", command)
	_, err = writeForSuccessResponse(dazeus, map[string]interface{}{
		"do":     "command",
		"params": append([]interface{}{command}, scopeSlice...),
	})

	if err != nil {
		return -1, err
	}

	handle := dazeus.lastHandle
	dazeus.lastHandle++
	dazeus.listeners[handle] = ldata

	return handle, nil
}

// Unsubscribe removes a subscription to a specific kind of event
func (dazeus *DaZeus) Unsubscribe(handle ListenerHandle) error {
	listener, ok := dazeus.listeners[handle]

	if !ok {
		return errors.New("No listener found")
	}
	delete(dazeus.listeners, handle)

	if listener.event != "COMMAND" {
		dazeus.logger.Printf("Removed event listener for events of type '%s'", listener.event)
		found := false
		for _, l := range dazeus.listeners {
			if l.event == listener.event {
				found = true
				break
			}
		}

		if !found {
			dazeus.logger.Printf("Unsubscribing to core events of type '%s'", listener.event)
			_, err := writeForSuccessResponse(dazeus, map[string]interface{}{
				"do":     "unsubscribe",
				"params": []string{string(listener.event)},
			})

			return err
		}
	} else {
		dazeus.logger.Printf("Removed command listener for commands of type '%s'", listener.command)
	}

	return nil
}

// Networks retrieves the networks the DaZeus core is connected to.
func (dazeus *DaZeus) Networks() ([]string, error) {
	resp, err := writeForSuccessResponse(dazeus, map[string]interface{}{
		"get": "networks",
	})
	if err != nil {
		return nil, err
	}

	return makeStringArray(resp["networks"])
}

// Channels lists the channels to which the bot is connected in the given network.
func (dazeus *DaZeus) Channels(network string) ([]string, error) {
	resp, err := writeForSuccessResponse(dazeus, map[string]interface{}{
		"get":    "channels",
		"params": []string{network},
	})
	if err != nil {
		return nil, err
	}

	return makeStringArray(resp["channels"])
}

// Join allows the bot to join a specific channel in some network
func (dazeus *DaZeus) Join(network string, channel string) error {
	_, err := writeForSuccessResponse(dazeus, map[string]interface{}{
		"do":     "join",
		"params": []string{network, channel},
	})

	return err
}

// Part allows the bot to leave a specific channel in some network.
func (dazeus *DaZeus) Part(network string, channel string) error {
	_, err := writeForSuccessResponse(dazeus, map[string]interface{}{
		"do":     "part",
		"params": []string{network, channel},
	})

	return err
}

// Message sends the given message to some channel in some network.
func (dazeus *DaZeus) Message(network string, channel string, message string) error {
	_, err := writeForSuccessResponse(dazeus, map[string]interface{}{
		"do":     "message",
		"params": []string{network, channel, message},
	})

	return err
}

// Action sends a CTCP action message to a channel in some network.
func (dazeus *DaZeus) Action(network string, channel string, message string) error {
	_, err := writeForSuccessResponse(dazeus, map[string]interface{}{
		"do":     "action",
		"params": []string{network, channel, message},
	})

	return err
}

// Notice sends a notice message to a channel in some network.
func (dazeus *DaZeus) Notice(network string, channel string, message string) error {
	_, err := writeForSuccessResponse(dazeus, map[string]interface{}{
		"do":     "notice",
		"params": []string{network, channel, message},
	})

	return err
}

// Ctcp sends a CTCP message to a channel in some network.
func (dazeus *DaZeus) Ctcp(network string, channel string, message string) error {
	_, err := writeForSuccessResponse(dazeus, map[string]interface{}{
		"do":     "ctcp",
		"params": []string{network, channel, message},
	})

	return err
}

// CtcpReply sends a CTCP reply message to a channel in some network.
func (dazeus *DaZeus) CtcpReply(network string, channel string, message string) error {
	_, err := writeForSuccessResponse(dazeus, map[string]interface{}{
		"do":     "ctcp_rep",
		"params": []string{network, channel, message},
	})

	return err
}

// Nick retrieves the nickname for the bot in a specific network.
func (dazeus *DaZeus) Nick(network string) (string, error) {
	resp, err := writeForSuccessResponse(dazeus, map[string]interface{}{
		"get":    "nick",
		"params": []string{network},
	})

	if err != nil {
		return "", err
	}

	fmt.Printf("Nicks resp %#v", resp)

	nick, ok := resp["nick"].(string)

	if !ok {
		return "", errors.New("No nick found in response")
	}

	return nick, nil
}

// GetConfig retrieves a config value.
func (dazeus *DaZeus) GetConfig(key string, group string) (string, error) {
	resp, err := writeForSuccessResponse(dazeus, map[string]interface{}{
		"get":    "config",
		"params": []string{group, key},
	})

	if err != nil {
		return "", err
	}

	value, ok := resp["value"].(string)

	if !ok {
		return "", errors.New("No value found in response")
	}

	return value, nil
}

// GetPluginConfig gets a config value for the plugin from the DaZeus core.
func (dazeus *DaZeus) GetPluginConfig(key string) (string, error) {
	return dazeus.GetConfig(key, "plugin")
}

// GetCoreConfig gets a config value for the DaZeus core.
func (dazeus *DaZeus) GetCoreConfig(key string) (string, error) {
	return dazeus.GetConfig(key, "core")
}

// HighlightCharacter gets the character used for highlighting the bot.
func (dazeus *DaZeus) HighlightCharacter() (string, error) {
	return dazeus.GetCoreConfig("highlight")
}

// GetProperty retrieves a property for a given scope.
func (dazeus *DaZeus) GetProperty(property string, scope Scope) (string, error) {
	var err error
	var resp map[string]interface{}

	if scope.IsAll() {
		resp, err = writeForSuccessResponse(dazeus, map[string]interface{}{
			"do":     "property",
			"params": []string{"get", property},
		})
	} else {
		resp, err = writeForSuccessResponse(dazeus, map[string]interface{}{
			"do":     "property",
			"scope":  scope.ToSlice(),
			"params": []string{"get", property},
		})
	}

	if err != nil {
		return "", err
	}

	value, ok := resp["value"].(string)

	if !ok {
		return "", errors.New("No value found in response")
	}

	return value, nil
}

// SetProperty sets a property to a string value for a given Scope.
func (dazeus *DaZeus) SetProperty(property string, value string, scope Scope) (err error) {
	if scope.IsAll() {
		_, err = writeForSuccessResponse(dazeus, map[string]interface{}{
			"do":     "property",
			"params": []string{"set", property, value},
		})
	} else {
		_, err = writeForSuccessResponse(dazeus, map[string]interface{}{
			"do":     "property",
			"scope":  scope.ToSlice(),
			"params": []string{"set", property, value},
		})
	}

	return
}

// UnsetProperty removes a property from the DaZeus core.
func (dazeus *DaZeus) UnsetProperty(property string, scope Scope) (err error) {
	if scope.IsAll() {
		_, err = writeForSuccessResponse(dazeus, map[string]interface{}{
			"do":     "property",
			"params": []string{"unset", property},
		})
	} else {
		_, err = writeForSuccessResponse(dazeus, map[string]interface{}{
			"do":     "property",
			"scope":  scope.ToSlice(),
			"params": []string{"unset", property},
		})
	}

	return
}

// PropertyKeys retrieves all keys matching a given prefix and scope.
func (dazeus *DaZeus) PropertyKeys(prefix string, scope Scope) ([]string, error) {
	var err error
	var resp map[string]interface{}

	if scope.IsAll() {
		resp, err = writeForSuccessResponse(dazeus, map[string]interface{}{
			"do":     "property",
			"params": []string{"keys", prefix},
		})
	} else {
		resp, err = writeForSuccessResponse(dazeus, map[string]interface{}{
			"do":     "property",
			"scope":  scope.ToSlice(),
			"params": []string{"keys", prefix},
		})
	}

	if err != nil {
		return nil, err
	}

	return makeStringArray(resp["keys"])
}

// HasPermission checks if a permission is given for the given scope.
func (dazeus *DaZeus) HasPermission(permission string, scope Scope, allow bool) (bool, error) {
	if scope.IsAll() {
		return false, errors.New("Will not check permission for universal scope")
	}

	resp, err := writeForSuccessResponse(dazeus, map[string]interface{}{
		"do":     "permission",
		"scope":  scope.ToSlice(),
		"params": []interface{}{"has", permission, allow},
	})

	if err != nil {
		return false, err
	}

	perm, ok := resp["has_permission"].(bool)
	if !ok {
		return false, errors.New("Did not retrieve permission from server")
	}
	return perm, nil
}

// SetPermission sets a permission for a given scope.
func (dazeus *DaZeus) SetPermission(permission string, scope Scope, allow bool) (err error) {
	if scope.IsAll() {
		return errors.New("Will not set permission for universal scope")
	}

	_, err = writeForSuccessResponse(dazeus, map[string]interface{}{
		"do":     "permission",
		"scope":  scope.ToSlice(),
		"params": []interface{}{"set", permission, allow},
	})
	return
}

// UnsetPermission removes a permission for some scope.
func (dazeus *DaZeus) UnsetPermission(permission string, scope Scope) (err error) {
	if scope.IsAll() {
		return errors.New("Will not remove permission for universal scope")
	}

	_, err = writeForSuccessResponse(dazeus, map[string]interface{}{
		"do":     "permission",
		"scope":  scope.ToSlice(),
		"params": []interface{}{"unset", permission},
	})
	return
}

// Whois sends a whois request for some nick in some network.
func (dazeus *DaZeus) Whois(network string, nick string) error {
	_, err := writeForSuccessResponse(dazeus, map[string]interface{}{
		"do":     "whois",
		"params": []string{network, nick},
	})

	return err
}

// Names sends a names request to some channel in some network, retrieving all nicks in that channel.
func (dazeus *DaZeus) Names(network string, channel string) error {
	_, err := writeForSuccessResponse(dazeus, map[string]interface{}{
		"do":     "names",
		"params": []string{network, channel},
	})

	return err
}

// Reply replies with a normal message to the correct channel.
func (dazeus *DaZeus) Reply(network string, channel string, sender string, message string, highlight bool) error {
	nick, err := dazeus.Nick(network)
	if err != nil {
		return err
	}

	if channel == nick {
		return dazeus.Message(network, sender, message)
	}

	if highlight {
		message = sender + ": " + message
	}

	return dazeus.Message(network, channel, message)
}

// ReplyNotice replies with a notice to the correct channel.
func (dazeus *DaZeus) ReplyNotice(network string, channel string, sender string, message string, highlight bool) error {
	nick, err := dazeus.Nick(network)
	if err != nil {
		return err
	}

	if channel == nick {
		return dazeus.Notice(network, sender, message)
	}

	if highlight {
		message = sender + ": " + message
	}

	return dazeus.Notice(network, channel, message)
}

// ReplyAction replies with a CTCP action to the correct channel.
func (dazeus *DaZeus) ReplyAction(network string, channel string, sender string, message string) error {
	nick, err := dazeus.Nick(network)
	if err != nil {
		return err
	}

	if channel == nick {
		return dazeus.Action(network, sender, message)
	}

	return dazeus.Action(network, channel, message)
}

// ReplyCtcpReply replies with a CTCP Reply to the correct channel.
func (dazeus *DaZeus) ReplyCtcpReply(network string, channel string, sender string, message string) error {
	nick, err := dazeus.Nick(network)
	if err != nil {
		return err
	}

	if channel == nick {
		return dazeus.CtcpReply(network, sender, message)
	}

	return dazeus.CtcpReply(network, channel, message)
}
