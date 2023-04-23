package utils

import (
	"fmt"
	"testing"

	"github.com/bmizerany/assert"
)

func TestPrefixMatch(t *testing.T) {
	pm := PrefixMatch{}
	pm.Build([]string{"/api", "/api/auth", "blog"})
	//pm.PPrint()
	//fmt.Printf("match(api) = %+v\n", pm.Match("api"))
	assert.Equal(t, pm.Match("/api"), []string{"/api"})
	assert.Equal(t, pm.Match("/api/index.html"), []string{"/api"})
	assert.Equal(t, pm.Match("/api/auth"), []string{"/api", "/api/auth"})
	assert.Equal(t, pm.Match("/api/auth/1234"), []string{"/api", "/api/auth"})
	//fmt.Printf("match(api/auth) = %+v\n", pm.Match("api/auth"))
	fmt.Printf("match(233) = %+v\n", pm.Match("233"))
}
