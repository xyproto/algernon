package utils

import (
	"log"
	"testing"

	"github.com/bmizerany/assert"
)

func TestKeywordTest(t *testing.T) {
	stripped, kwMap := ExtractKeywords([]byte(`<!--
title: test
theme: fest
-->
# Headline
text
`), []string{"title", "theme", "beard"})
	log.Println(stripped, kwMap)
	assert.Equal(t, nil, nil)
}
