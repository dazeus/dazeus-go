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
