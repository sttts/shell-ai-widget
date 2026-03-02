package main

import "testing"

func TestResolveShell(t *testing.T) {
	tests := []struct {
		name     string
		cliShell string
		shellEnv string
		want     string
	}{
		{
			name:     "cli override wins",
			cliShell: "fish",
			shellEnv: "/bin/zsh",
			want:     "fish",
		},
		{
			name:     "cli override is trimmed",
			cliShell: "  fish  ",
			shellEnv: "/bin/zsh",
			want:     "fish",
		},
		{
			name:     "uses shell env basename",
			cliShell: "",
			shellEnv: "/bin/zsh",
			want:     "zsh",
		},
		{
			name:     "uses longer shell env path basename",
			cliShell: "",
			shellEnv: "/opt/homebrew/bin/fish",
			want:     "fish",
		},
		{
			name:     "empty shell env falls back",
			cliShell: "",
			shellEnv: "",
			want:     "zsh",
		},
		{
			name:     "whitespace shell env falls back",
			cliShell: "",
			shellEnv: "   ",
			want:     "zsh",
		},
		{
			name:     "root shell env falls back",
			cliShell: "",
			shellEnv: "/",
			want:     "zsh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveShell(tt.cliShell, tt.shellEnv)
			if got != tt.want {
				t.Fatalf("resolveShell(%q, %q) = %q, want %q", tt.cliShell, tt.shellEnv, got, tt.want)
			}
		})
	}
}
