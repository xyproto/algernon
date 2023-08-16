package utils

import (
	"bytes"
	"testing"
)

func TestExtractKW1(t *testing.T) {
	_, kwMap := ExtractKeywords([]byte(`<!--
title: test
theme: fest
-->
# Headline
text
`), []string{"title", "theme", "beard"})

	if !bytes.Equal(kwMap["title"], []byte("test")) {
		t.Errorf("Expected 'test', got '%s'", kwMap["title"])
	}

	if !bytes.Equal(kwMap["theme"], []byte("fest")) {
		t.Errorf("Expected 'fest', got '%s'", kwMap["theme"])
	}

	if _, ok := kwMap["beard"]; ok {
		t.Errorf("Expected 'beard' key to be absent, but it exists.")
	}

	if kwMap["beard"] != nil && len(kwMap["beard"]) != 0 {
		t.Errorf("Expected empty byte slice, got '%s'", kwMap["beard"])
	}
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

	if !bytes.Equal(kwMap["title"], []byte("Best Title")) {
		t.Errorf("Expected 'Best Title', got '%s'", kwMap["title"])
	}

	if !bytes.Equal(kwMap["weather"], []byte("nice")) {
		t.Errorf("Expected 'nice', got '%s'", kwMap["weather"])
	}

	if !bytes.Equal(kwMap["boat"], []byte("missing")) {
		t.Errorf("Expected 'missing', got '%s'", kwMap["boat"])
	}

	if _, ok := kwMap["horse"]; ok {
		t.Errorf("Expected 'horse' key to be absent, but it exists.")
	}

	if kwMap["horse"] != nil && len(kwMap["horse"]) != 0 {
		t.Errorf("Expected empty byte slice, got '%s'", kwMap["horse"])
	}
}
