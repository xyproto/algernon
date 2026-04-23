package themes

import (
	_ "embed"
)

// Latest version of MUI: https://www.muicss.com/

// MUICSS is from http://cdn.muicss.com/mui-0.9.39-rc1/css/mui.min.css
//
//go:embed assets/mui.css
var MUICSS string

// MUIJS is from http://cdn.muicss.com/mui-0.9.39-rc1/js/mui.min.js
//
//go:embed assets/mui.js
var MUIJS string

// MaterialHead enables the Material style by adding CSS and JS tags that can go in a header
func MaterialHead() string {
	return "<style>" + MUICSS + "</style><script type=\"javascript\">" + MUIJS + "</script>"
}
