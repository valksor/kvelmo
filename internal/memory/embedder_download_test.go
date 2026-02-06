package memory

import "testing"

func TestIsEmbedderAvailable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		goos   string
		goarch string
		want   bool
	}{
		{name: "linux amd64", goos: "linux", goarch: "amd64", want: true},
		{name: "linux arm64", goos: "linux", goarch: "arm64", want: true},
		{name: "darwin amd64", goos: "darwin", goarch: "amd64", want: true},
		{name: "darwin arm64", goos: "darwin", goarch: "arm64", want: true},
		{name: "windows amd64", goos: "windows", goarch: "amd64", want: false},
		{name: "windows arm64", goos: "windows", goarch: "arm64", want: false},
		{name: "linux 386", goos: "linux", goarch: "386", want: false},
		{name: "darwin 386", goos: "darwin", goarch: "386", want: false},
		{name: "freebsd amd64", goos: "freebsd", goarch: "amd64", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := isEmbedderAvailable(tc.goos, tc.goarch)
			if got != tc.want {
				t.Fatalf("isEmbedderAvailable(%q, %q) = %v, want %v", tc.goos, tc.goarch, got, tc.want)
			}
		})
	}
}
