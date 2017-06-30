package parseutil

import "fmt"

func ParseStringList(maybeStringList []interface{}) ([]string, error) {
	var strings []string
	for _, v := range maybeStringList {
		switch v.(type) {
		case string:
			strings = append(strings, v.(string))
		default:
			return nil, fmt.Errorf("not list of same type")
		}
	}

	return strings, nil
}
