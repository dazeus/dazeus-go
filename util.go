package dazeus

import "errors"

// makeStringArray creates an array of strings from a value in the json message
func makeStringArray(fieldValue interface{}) ([]string, error) {
	arr, ok := fieldValue.([]interface{})
	strs := make([]string, 0)

	if !ok {
		return nil, errors.New("Could not find expected array")
	}

	for _, val := range arr {
		str, ok := val.(string)
		if !ok {
			return nil, errors.New("Found non-string value in array")
		}

		strs = append(strs, str)
	}

	return strs, nil
}

func makeReplier(dazeus *DaZeus, network string, channel string, sender string) Replier {
	return func(message string, replyType replyFunction, highlight bool) error {
		switch replyType {
		case ReplyMessage:
			return dazeus.ReplyMessage(network, channel, sender, message, highlight)
		case ReplyAction:
			return dazeus.ReplyAction(network, channel, sender, message)
		case ReplyNotice:
			return dazeus.ReplyNotice(network, channel, sender, message, highlight)
		case ReplyCtcpReply:
			return dazeus.ReplyCtcpReply(network, channel, sender, message)
		default:
			return errors.New("Unknown reply function requested")
		}
	}
}
