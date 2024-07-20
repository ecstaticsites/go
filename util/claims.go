package util

import (
	"fmt"
	"math"
)

// lots of parsing and checking in order to get the list of authorized
// pull zone IDs from a claims dict in a JWT, that's this
func GetZoneIdsFromClaims(claims map[string]interface{}) ([]int, error) {

	metadata, found1 := claims["app_metadata"]
	if !found1 {
		return nil, fmt.Errorf("No 'app_metadata' found in JWT claims")
	}

	metadataMap, ok1 := metadata.(map[string]interface{})
	if !ok1 {
		return nil, fmt.Errorf("Claims 'app_metadata' could not be parsed as map")
	}

	zones, found2 := metadataMap["zones"]
	if !found2 {
		// return early with no error -- just means they've created no zones yet
		return nil, nil
	}

	zonesArray, ok2 := zones.([]interface{})
	if !ok2 {
		return nil, fmt.Errorf("Metadata 'zones' field could not be parsed as array")
	}

	zoneIntArray := []int{}
	for _, zone := range zonesArray {
		// not sure why this comes in in float-form? It's just an int
		zoneFloat, ok3 := zone.(float64)
		if !ok3 {
			return nil, fmt.Errorf("Item in metadata 'zones' array could not be parsed")
		}
		zoneInt := int(math.Round(zoneFloat))
		zoneIntArray = append(zoneIntArray, zoneInt)
	}

	return zoneIntArray, nil
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
