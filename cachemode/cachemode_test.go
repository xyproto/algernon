package cachemode

import "testing"

func TestNew(t *testing.T) {
	tests := []struct {
		input string
		want  Setting
	}{
		{"on", On},
		{"everything", On},
		{"all", On},
		{"1", On},
		{"enabled", On},
		{"yes", On},
		{"enable", On},
		{"off", Off},
		{"disabled", Off},
		{"0", Off},
		{"no", Off},
		{"disable", Off},
		{"production", Production},
		{"prod", Production},
		{"images", Images},
		{"image", Images},
		{"small", Small},
		{"64k", Small},
		{"64KB", Small},
		{"dev", Default},
		{"default", Default},
		{"unset", Default},
		{"anything_else", Default},
	}
	for _, tt := range tests {
		got := New(tt.input)
		if got != tt.want {
			t.Errorf("New(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		setting Setting
		want    string
	}{
		{On, "On"},
		{Off, "Off"},
		{Development, "Development"},
		{Production, "Production"},
		{Images, "Images"},
		{Small, "Small"},
		{Unset, "unset"},
	}
	for _, tt := range tests {
		got := tt.setting.String()
		if got != tt.want {
			t.Errorf("Setting(%d).String() = %q, want %q", tt.setting, got, tt.want)
		}
	}
}
