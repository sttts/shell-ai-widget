package tui

import "testing"

func TestWrapBuffer(t *testing.T) {
	tests := []struct {
		name   string
		buffer string
		width  int
		want   []string
	}{
		{
			name:   "fits_one_line",
			buffer: "echo hi",
			width:  80,
			want:   []string{"❯ echo hi"},
		},
		{
			name:   "wraps_multiple_lines",
			buffer: "0123456789abcdef",
			width:  10,
			want:   []string{"❯ 01234567", "  89abcdef"},
		},
		{
			name:   "preserves_leading_spaces",
			buffer: "  leading",
			width:  20,
			want:   []string{"❯   leading"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapBuffer(tt.buffer, tt.width)
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("line %d = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
