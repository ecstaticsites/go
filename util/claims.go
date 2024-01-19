package util

import (
	"fmt"
)

// lots of parsing and checking in order to get the list of authorized
// hostnames from a claims dict in a JWT, that's this
func GetHostnamesFromClaims(claims map[string]interface{}) ([]string, error) {

	metadata, found1 := claims["app_metadata"]
	if !found1 {
		return nil, fmt.Errorf("No 'app_metadata' found in JWT claims")
	}

	metadataMap, ok1 := metadata.(map[string]interface{})
	if !ok1 {
		return nil, fmt.Errorf("Claims 'app_metadata' could not be parsed as map")
	}

	hostnames, found2 := metadataMap["hostnames"]
	if !found2 {
		// return early with no error -- just means they've created no sites yet
		return nil, nil
	}

	hostnamesArray, ok2 := hostnames.([]interface{})
	if !ok2 {
		return nil, fmt.Errorf("Metadata 'hostnames' field could not be parsed as array")
	}

	hostnamesStringArray := []string{}
	for _, name := range hostnamesArray {
		nameString, ok3 := name.(string)
		if !ok3 {
			return nil, fmt.Errorf("Item in metadata 'hostnames' array could not be parsed as string")
		}
		hostnamesStringArray = append(hostnamesStringArray, nameString)
	}

	return hostnamesStringArray, nil
}

// similar to the above, but gets the straightforward "readonly" field
func GetReadonlyFromClaims(claims map[string]interface{}) (bool, error) {

	metadata, found1 := claims["app_metadata"]
	if !found1 {
		return false, fmt.Errorf("No 'app_metadata' found in JWT claims")
	}

	metadataMap, ok1 := metadata.(map[string]interface{})
	if !ok1 {
		return false, fmt.Errorf("Claims 'app_metadata' could not be parsed as map")
	}

	readonly, found2 := metadataMap["readonly"]
	if !found2 {
		// return early with no error -- only enforce readonly if field is present
		return false, nil
	}

	readonlyString, ok2 := readonly.(bool)
	if !ok2 {
		return false, fmt.Errorf("Metadata 'readonly' field could not be parsed as bool")
	}

	return readonlyString, nil
}
