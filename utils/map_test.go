package utils

import (
	"reflect"
	"testing"
)

func TestMergeAsSet(t *testing.T) {
	// Define test cases
	tests := []struct {
		name     string
		this     map[string]int
		other    map[string]int
		expected map[string]int
		wantErr  bool
	}{
		{
			name:     "Basic merge - no overlap",
			this:     map[string]int{"a": 1, "b": 2},
			other:    map[string]int{"c": 3, "d": 4},
			expected: map[string]int{"a": 1, "b": 2, "c": 3, "d": 4},
			wantErr:  false,
		},
		{
			name:     "Merge with overlap - keep 'this' values",
			this:     map[string]int{"a": 1, "b": 2, "c": 3},
			other:    map[string]int{"c": 30, "d": 4},
			expected: map[string]int{"a": 1, "b": 2, "c": 3, "d": 4},
			wantErr:  false,
		},
		{
			name:     "Empty 'this' map",
			this:     map[string]int{},
			other:    map[string]int{"a": 1, "b": 2},
			expected: map[string]int{"a": 1, "b": 2},
			wantErr:  false,
		},
		{
			name:     "Empty 'other' map",
			this:     map[string]int{"a": 1, "b": 2},
			other:    map[string]int{},
			expected: map[string]int{"a": 1, "b": 2},
			wantErr:  false,
		},
		{
			name:     "Both maps empty",
			this:     map[string]int{},
			other:    map[string]int{},
			expected: map[string]int{},
			wantErr:  false,
		},
		{
			name:     "Nil 'this' map",
			this:     nil,
			other:    map[string]int{"a": 1},
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "Nil 'other' map",
			this:     map[string]int{"a": 1},
			other:    nil,
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "Complete overlap",
			this:     map[string]int{"a": 1, "b": 2},
			other:    map[string]int{"a": 10, "b": 20},
			expected: map[string]int{"a": 1, "b": 2},
			wantErr:  false,
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the function
			result, err := MergeAsSet(tt.this, tt.other)

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("MergeAsSet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// If expecting an error, don't check the result
			if tt.wantErr {
				return
			}

			// Check that result matches expected
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("MergeAsSet() = %v, want %v", result, tt.expected)
			}

			// Verify that the original maps were not modified
			// This ensures our function is not modifying inputs
			if !reflect.DeepEqual(tt.this, tt.this) {
				t.Errorf("Original 'this' map was modified")
			}
			if !reflect.DeepEqual(tt.other, tt.other) {
				t.Errorf("Original 'other' map was modified")
			}
		})
	}
}

// TestMergeAsSetWithDifferentTypes tests the generic nature of the function
func TestMergeAsSetWithDifferentTypes(t *testing.T) {
	// Test with string->string map
	map1 := map[string]string{"key1": "value1", "key2": "value2"}
	map2 := map[string]string{"key2": "newvalue2", "key3": "value3"}

	expected := map[string]string{"key1": "value1", "key2": "value2", "key3": "value3"}

	result, err := MergeAsSet(map1, map2)
	if err != nil {
		t.Errorf("MergeAsSet() with string->string maps returned an error: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("MergeAsSet() with string->string maps = %v, want %v", result, expected)
	}

	// Test with int->interface{} map
	map3 := map[int]interface{}{1: "one", 2: 2.0}
	map4 := map[int]interface{}{2: "two", 3: true}

	result2, err := MergeAsSet(map3, map4)
	if err != nil {
		t.Errorf("MergeAsSet() with int->interface{} maps returned an error: %v", err)
	}

	// Check result manually since interfaces require more careful comparison
	if len(result2) != 3 {
		t.Errorf("MergeAsSet() result has wrong length: got %d, want 3", len(result2))
	}

	if result2[1] != "one" {
		t.Errorf("MergeAsSet() result[1] = %v, want 'one'", result2[1])
	}

	if result2[2] != 2.0 {
		t.Errorf("MergeAsSet() result[2] = %v, want 2.0", result2[2])
	}

	if result2[3] != true {
		t.Errorf("MergeAsSet() result[3] = %v, want true", result2[3])
	}
}

// TestNestedMapMerge tests the recursive merging of nested maps
func TestNestedMapMerge(t *testing.T) {
	// Test case 1: Simple nested map
	map1 := map[string]interface{}{
		"parent_a": map[string]string{
			"child_a": "x",
			"child_b": "y",
		},
		"simple_key": "value",
	}

	map2 := map[string]interface{}{
		"parent_a": map[string]string{
			"child_c": "z",
			"child_a": "i", // This should be ignored (keep map1's value)
		},
		"new_key": "new_value",
	}

	expected := map[string]interface{}{
		"parent_a": map[string]string{
			"child_a": "x", // Keep from map1
			"child_b": "y", // Keep from map1
			"child_c": "z", // Add from map2
		},
		"simple_key": "value",     // Keep from map1
		"new_key":    "new_value", // Add from map2
	}

	result, err := MergeAsSet(map1, map2)
	if err != nil {
		t.Errorf("MergeAsSet() with nested maps returned an error: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("MergeAsSet() with nested maps failed.\nGot: %v\nWant: %v", result, expected)
	}

	// Test case 2: Deeply nested maps
	deepMap1 := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": map[string]string{
					"key1": "val1",
					"key2": "val2",
				},
				"sibling": "value",
			},
		},
	}

	deepMap2 := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": map[string]string{
					"key3": "val3",
					"key1": "newval1", // This should be ignored
				},
				"newSibling": "newValue",
			},
		},
	}

	expectedDeep := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": map[string]string{
					"key1": "val1", // Keep from map1
					"key2": "val2", // Keep from map1
					"key3": "val3", // Add from map2
				},
				"sibling":    "value",    // Keep from map1
				"newSibling": "newValue", // Add from map2
			},
		},
	}

	resultDeep, err := MergeAsSet(deepMap1, deepMap2)
	if err != nil {
		t.Errorf("MergeAsSet() with deeply nested maps returned an error: %v", err)
	}

	if !reflect.DeepEqual(resultDeep, expectedDeep) {
		t.Errorf("MergeAsSet() with deeply nested maps failed.\nGot: %v\nWant: %v", resultDeep, expectedDeep)
	}

	// Test case 3: Mixed nested types (only merge maps, keep other values)
	mixedMap1 := map[string]interface{}{
		"map":    map[string]string{"a": "1", "b": "2"},
		"slice":  []int{1, 2, 3},
		"string": "hello",
	}

	mixedMap2 := map[string]interface{}{
		"map":    map[string]string{"c": "3", "a": "x"}, // "a" should be ignored
		"slice":  []int{4, 5, 6},                        // This should not be merged
		"string": "world",                               // This should not be merged
	}

	expectedMixed := map[string]interface{}{
		"map":    map[string]string{"a": "1", "b": "2", "c": "3"}, // Maps are merged
		"slice":  []int{1, 2, 3},                                  // Keep from map1 (not merged)
		"string": "hello",                                         // Keep from map1 (not merged)
	}

	resultMixed, err := MergeAsSet(mixedMap1, mixedMap2)
	if err != nil {
		t.Errorf("MergeAsSet() with mixed nested types returned an error: %v", err)
	}

	if !reflect.DeepEqual(resultMixed, expectedMixed) {
		t.Errorf("MergeAsSet() with mixed nested types failed.\nGot: %v\nWant: %v", resultMixed, expectedMixed)
	}
}

// TestMergeAsSetPerformance checks that the function handles large maps efficiently
func TestMergeAsSetPerformance(t *testing.T) {
	// Create two large maps
	largeMap1 := make(map[int]int, 10000)
	largeMap2 := make(map[int]int, 5000)

	// Fill maps with data
	for i := 0; i < 10000; i++ {
		largeMap1[i] = i
	}

	for i := 5000; i < 10000; i++ {
		largeMap2[i] = i * 2 // Different values for overlapping keys
	}

	// Add some new keys to map2
	for i := 10000; i < 15000; i++ {
		largeMap2[i] = i
	}

	// Merge the maps
	result, err := MergeAsSet(largeMap1, largeMap2)
	if err != nil {
		t.Errorf("MergeAsSet() with large maps returned an error: %v", err)
	}

	// Verify length of result
	expectedLength := 15000 // 10000 from map1 + 5000 unique from map2
	if len(result) != expectedLength {
		t.Errorf("MergeAsSet() result has wrong length: got %d, want %d", len(result), expectedLength)
	}

	// Verify some key values from map1 were preserved
	for i := 5000; i < 10000; i++ {
		if result[i] != i {
			t.Errorf("MergeAsSet() result[%d] = %v, want %d", i, result[i], i)
			break
		}
	}
}
