package dazeus

import "errors"

// Scope for which a request is sent
type Scope struct {
	Network, Receiver, Sender *string
}

// NewUniversalScope returns a universal scope
func NewUniversalScope() Scope {
	return Scope{
		Network:  nil,
		Receiver: nil,
		Sender:   nil,
	}
}

// NewNetworkScope returns a scope limited to a network
func NewNetworkScope(network string) Scope {
	return Scope{
		Network:  &network,
		Receiver: nil,
		Sender:   nil,
	}
}

// NewReceiverScope returns a scope limited to a network and a specific receiver (channel) in that network
func NewReceiverScope(network string, receiver string) Scope {
	return Scope{
		Network:  &network,
		Receiver: &receiver,
		Sender:   nil,
	}
}

// NewSenderScope returns a scope limited to a network and a specific sender (user) in that network
func NewSenderScope(network string, sender string) Scope {
	return Scope{
		Network:  &network,
		Receiver: nil,
		Sender:   &sender,
	}
}

// NewScope returns a scope limited on a specific sender and receiver combination in a specific network
func NewScope(network string, receiver string, sender string) Scope {
	return Scope{
		Network:  &network,
		Receiver: &receiver,
		Sender:   &sender,
	}
}

// IsAll indicates if this scope is global
func (scope Scope) IsAll() bool {
	return scope.Network == nil && scope.Receiver == nil && scope.Sender == nil
}

// ToSlice returns a slice for usage with permissions and properties
func (scope Scope) ToSlice() []string {
	s := make([]string, 0)
	if scope.Network != nil {
		s = append(s, *scope.Network)
		if scope.Receiver != nil {
			s = append(s, *scope.Receiver)
			if scope.Sender != nil {
				s = append(s, *scope.Sender)
			}
		}
	}

	return s
}

// ToCommandSlice returns a slice for usage when sending with a command subscription
func (scope Scope) ToCommandSlice() ([]interface{}, error) {
	s := make([]interface{}, 0)

	if scope.Receiver != nil && scope.Sender != nil {
		return s, errors.New("")
	}

	if scope.Network != nil {
		s = append(s, *scope.Network)

		if scope.Receiver != nil {
			s = append(s, false)
			s = append(s, *scope.Receiver)
		}

		if scope.Sender != nil {
			s = append(s, false)
			s = append(s, *scope.Sender)
		}
	}

	return s, nil
}
