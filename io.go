package dazeus

import (
	"encoding/json"
	"errors"
	"strconv"
)

func checkMessage(dazeus *DaZeus) (bool, int, int) {
	var offset, messageLen int

	for offset < dazeus.buffer.Len() {
		curr := dazeus.buffer.Bytes()[offset]
		if curr >= '0' && curr <= '9' {
			messageLen *= 10
			messageLen += int(curr - '0')
			offset++
		} else if curr == '\n' || curr == '\r' {
			offset++
		} else {
			break
		}
	}

	if messageLen > 0 && dazeus.buffer.Len() >= offset+messageLen {
		return true, offset, messageLen
	}
	return false, 0, 0
}

func read(dazeus *DaZeus) (Message, error) {
	for {
		hasMessage, offset, messageLen := checkMessage(dazeus)
		if hasMessage {
			discard := make([]byte, offset)
			bytesRead, err := dazeus.buffer.Read(discard)

			if err != nil {
				return nil, err
			}

			if bytesRead != offset {
				return nil, errors.New("Could not read expected number of offset bytes")
			}

			message := make([]byte, messageLen)
			bytesRead, err = dazeus.buffer.Read(message)

			if err != nil {
				return nil, err
			}

			if bytesRead != messageLen {
				return nil, errors.New("Could not read expected message from buffer")
			}

			dazeus.logger.Printf("Received message from core: %s", message)

			msg := make(map[string]interface{})
			err = json.Unmarshal(message, &msg)

			if err != nil {
				return nil, err
			}

			return msg, nil
		}

		next := make([]byte, 1024)
		bytesRead, err := dazeus.conn.Read(next)

		if err != nil {
			return nil, err
		}

		bytesWritten, err := dazeus.buffer.Write(next[0:bytesRead])

		if err != nil {
			return nil, err
		}

		if bytesWritten != bytesRead {
			return nil, errors.New("Could not write bytes from socket to buffer")
		}
	}
}

func write(dazeus *DaZeus, message Message) error {
	bytes, err := json.Marshal(message)
	dazeus.logger.Printf("Sending message to core: %s", bytes)

	if err != nil {
		return err
	}

	msglen := []byte(strconv.Itoa(len(bytes)))
	tosend := append(msglen, bytes...)

	bytesWritten, err := dazeus.conn.Write(tosend)

	if err != nil {
		return err
	}

	if bytesWritten != len(tosend) {
		return errors.New("Could not write complete message to socket")
	}

	return nil
}

func waitForResponse(dazeus *DaZeus) (Message, error) {
	for {
		msg, err := read(dazeus)

		if err != nil {
			return nil, err
		}

		if msg["event"] != nil {
			dazeus.callDepth++
			err = handleEvent(dazeus, msg)
			dazeus.callDepth--

			if err != nil {
				return nil, err
			}

			// check for buffered messages
			if len(dazeus.responseQueue) > dazeus.callDepth {
				msg = dazeus.responseQueue[0]
				dazeus.responseQueue = dazeus.responseQueue[1:]

				return msg, nil
			}
		} else {
			if len(dazeus.responseQueue) == dazeus.callDepth {
				return msg, nil
			}

			// this one is for one call level above
			dazeus.responseQueue = append(dazeus.responseQueue, msg)
		}
	}
}

func waitForSuccessResponse(dazeus *DaZeus) (Message, error) {
	response, err := waitForResponse(dazeus)

	if err != nil {
		return nil, err
	}

	if response["success"] == nil {
		return nil, errors.New("No success field found")
	}

	success, ok := response["success"].(bool)
	if !ok {
		return nil, errors.New("No boolean in success field")
	}

	if !success {
		return nil, errors.New("Server responded with failure")
	}

	return response, nil
}

func writeForSuccessResponse(dazeus *DaZeus, message Message) (Message, error) {
	err := write(dazeus, message)
	if err != nil {
		return nil, err
	}

	resp, err := waitForSuccessResponse(dazeus)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func waitForEvent(dazeus *DaZeus) error {
	msg, err := read(dazeus)

	if err != nil {
		return err
	}

	if msg["event"] == nil {
		return errors.New("Unexpected non-event message retrieved")
	}

	err = handleEvent(dazeus, msg)
	if err != nil {
		return err
	}

	return nil
}
