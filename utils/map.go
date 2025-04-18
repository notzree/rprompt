package utils

import (
	"errors"
	"reflect"
)

// MergeAsSet merges two maps treating them as sets, where only keys matter
// and the first value for each key is preserved, recursively handling nested maps.
//
// Parameters:
//   - this: The base map to merge into (or use as primary source)
//   - other: The secondary map to merge from
//
// Returns:
//   - A new map containing all unique keys from both input maps with nested maps merged recursively
//   - An error if either input map is nil
func MergeAsSet[K comparable, V any](this, other map[K]V) (map[K]V, error) {
	// Validate inputs
	if this == nil || other == nil {
		return nil, errors.New("cannot merge nil maps")
	}

	// Create a new map with capacity to hold all potential entries
	// This avoids potential rehashing operations during insertion
	result := make(map[K]V, len(this)+len(other))

	// Copy entries from the first map
	for k, v := range this {
		result[k] = v
	}

	// Process entries from the second map
	for k, v := range other {
		if existingVal, exists := result[k]; exists {
			// If the key already exists in result, check if both values are maps
			// If they are, merge them recursively
			mergedValue, handleNestedMap := tryMergeNestedMaps(existingVal, v)
			if handleNestedMap {
				result[k] = mergedValue.(V)
			}
			// If not maps, keep the first map's value (already in result)
		} else {
			// Key doesn't exist in result, add it
			result[k] = v
		}
	}

	return result, nil
}

// tryMergeNestedMaps attempts to merge two values if they are both maps.
// Returns the merged result and a boolean indicating whether a merge was performed.
func tryMergeNestedMaps(val1, val2 interface{}) (interface{}, bool) {
	// Use reflection to check if both values are maps
	v1 := reflect.ValueOf(val1)
	v2 := reflect.ValueOf(val2)

	// Check if both values are maps
	if v1.Kind() == reflect.Map && v2.Kind() == reflect.Map {
		// Check if the maps have the same key type
		if v1.Type().Key() != v2.Type().Key() {
			return val1, false // Keep original if key types don't match
		}

		// Create a new map of the same type as val1
		resultMap := reflect.MakeMap(v1.Type())

		// Copy all entries from the first map
		for _, key := range v1.MapKeys() {
			resultMap.SetMapIndex(key, v1.MapIndex(key))
		}

		// Process entries from the second map
		for _, key := range v2.MapKeys() {
			// Check if the key exists in the first map
			if v1Key := v1.MapIndex(key); v1Key.IsValid() {
				// If both values at this key are maps, recurse
				v1Val := v1Key.Interface()
				v2Val := v2.MapIndex(key).Interface()

				mergedVal, wasMerged := tryMergeNestedMaps(v1Val, v2Val)
				if wasMerged {
					resultMap.SetMapIndex(key, reflect.ValueOf(mergedVal))
				}
				// If not maps, keep the first map's value (already in resultMap)
			} else {
				// Key doesn't exist in first map, add from second map
				resultMap.SetMapIndex(key, v2.MapIndex(key))
			}
		}

		return resultMap.Interface(), true
	}

	// If values aren't both maps, keep the first value
	return val1, false
}

// uniqueStrings returns a slice with duplicate strings removed
func UniqueString(items []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}
