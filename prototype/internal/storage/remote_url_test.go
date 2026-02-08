package storage

import "testing"

func TestSanitizeRemoteURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "empty",
			in:   "",
			want: "",
		},
		{
			name: "https without credentials",
			in:   "https://github.com/user/repo.git",
			want: "https://github.com/user/repo.git",
		},
		{
			name: "https token in userinfo",
			in:   "https://ghp_secret123@github.com/user/repo.git",
			want: "https://github.com/user/repo.git",
		},
		{
			name: "https user and password",
			in:   "https://user:ghp_secret123@github.com/user/repo.git",
			want: "https://github.com/user/repo.git",
		},
		{
			name: "ssh git user unchanged",
			in:   "git@github.com:user/repo.git",
			want: "git@github.com:user/repo.git",
		},
		{
			name: "schemeless token userinfo",
			in:   "ghp_secret123@github.com/user/repo.git",
			want: "github.com/user/repo.git",
		},
		{
			name: "schemeless oauth2 userinfo",
			in:   "oauth2@gitlab.com/group/repo.git",
			want: "gitlab.com/group/repo.git",
		},
		{
			name: "project ID with token (dashes instead of slashes)",
			in:   "ghp_secret123@github.com-user-repo",
			want: "github.com-user-repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeRemoteURL(tt.in)
			if got != tt.want {
				t.Errorf("SanitizeRemoteURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
