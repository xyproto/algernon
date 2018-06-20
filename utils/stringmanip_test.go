package utils

import (
	"testing"

	"github.com/bmizerany/assert"
)

func TestExtractKW1(t *testing.T) {
	_, kwMap := ExtractKeywords([]byte(`<!--
title: test
theme: fest
-->
# Headline
text
`), []string{"title", "theme", "beard"})
	assert.Equal(t, string(kwMap["title"]), "test")
	assert.Equal(t, string(kwMap["theme"]), "fest")
	_, ok := kwMap["beard"]
	assert.Equal(t, ok, false)
	assert.Equal(t, string(kwMap["beard"]), "")
}

func TestExtractKW2(t *testing.T) {
	data := `
	% Best Title
	<!--
	weather: nice
	-->
	<!-- boat: missing -->
	<!-- horse: ignored -->

	# Page contents goes here

	And also here, but this should be ignored:
	title: nope
	weather: nope
	boat: nope

	All done.
	`
	_, kwMap := ExtractKeywords([]byte(data), []string{"title", "weather", "boat"})
	assert.Equal(t, string(kwMap["title"]), "Best Title")
	assert.Equal(t, string(kwMap["weather"]), "nice")
	assert.Equal(t, string(kwMap["boat"]), "missing")
	_, ok := kwMap["horse"]
	assert.Equal(t, ok, false)
	assert.Equal(t, string(kwMap["horse"]), "")
}
