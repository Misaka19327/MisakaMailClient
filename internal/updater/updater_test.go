package updater

import "testing"

func TestToggleVPrefix(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"0.5.0", "v0.5.0"},
		{"v0.5.0", "0.5.0"},
		{"V0.5.0", "0.5.0"},
		{"  v0.5.0  ", "0.5.0"},
		{"1.2.3-beta", "v1.2.3-beta"},
		{"v1.2.3-beta", "1.2.3-beta"},
	}
	for _, tc := range tests {
		if got := toggleVPrefix(tc.in); got != tc.want {
			t.Errorf("toggleVPrefix(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
