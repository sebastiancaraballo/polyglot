package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestIsConfirmKey(t *testing.T) {
	tests := []struct {
		name string
		msg  tea.KeyPressMsg
		want bool
	}{
		{name: "enter", msg: tea.KeyPressMsg{Code: tea.KeyEnter}, want: true},
		{name: "space", msg: tea.KeyPressMsg{Code: tea.KeySpace}, want: true},
		{name: "q", msg: tea.KeyPressMsg{Code: 'q', Text: "q"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsConfirmKey(tt.msg); got != tt.want {
				t.Fatalf("IsConfirmKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
