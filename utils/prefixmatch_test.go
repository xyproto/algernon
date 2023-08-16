package utils

import (
	"fmt"
	"testing"
)

func TestPrefixMatch(t *testing.T) {
	pm := PrefixMatch{}
	pm.Build([]string{"/api", "/api/auth", "blog"})

	expected1 := []string{"/api"}
	if res := pm.Match("/api"); !sliceEqual(res, expected1) {
		t.Errorf("Expected %v, got %v for input '/api'", expected1, res)
	}

	expected2 := []string{"/api"}
	if res := pm.Match("/api/index.html"); !sliceEqual(res, expected2) {
		t.Errorf("Expected %v, got %v for input '/api/index.html'", expected2, res)
	}

	expected3 := []string{"/api", "/api/auth"}
	if res := pm.Match("/api/auth"); !sliceEqual(res, expected3) {
		t.Errorf("Expected %v, got %v for input '/api/auth'", expected3, res)
	}

	expected4 := []string{"/api", "/api/auth"}
	if res := pm.Match("/api/auth/1234"); !sliceEqual(res, expected4) {
		t.Errorf("Expected %v, got %v for input '/api/auth/1234'", expected4, res)
	}

	fmt.Printf("match(233) = %+v\n", pm.Match("233"))
}

// sliceEqual checks if two string slices are equal
func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
