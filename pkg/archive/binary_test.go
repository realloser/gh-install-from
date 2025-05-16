package archive

import "testing"

func TestIsBinaryForPlatform(t *testing.T) {
	tests := []struct {
		name     string
		binary   string
		platform string
		want     bool
	}{
		{
			name:     "exact match",
			binary:   "app-darwin-amd64",
			platform: "darwin_amd64",
			want:     true,
		},
		{
			name:     "compressed tar.gz",
			binary:   "app-darwin-amd64.tar.gz",
			platform: "darwin_amd64",
			want:     true,
		},
		{
			name:     "compressed zip",
			binary:   "app-darwin-amd64.zip",
			platform: "darwin_amd64",
			want:     true,
		},
		{
			name:     "alternative os name",
			binary:   "app-macos-amd64",
			platform: "darwin_amd64",
			want:     true,
		},
		{
			name:     "alternative arch name",
			binary:   "app-darwin-x86_64",
			platform: "darwin_amd64",
			want:     true,
		},
		{
			name:     "windows exe",
			binary:   "app.exe",
			platform: "windows_amd64",
			want:     true,
		},
		{
			name:     "wrong os",
			binary:   "app-linux-amd64",
			platform: "darwin_amd64",
			want:     false,
		},
		{
			name:     "wrong arch",
			binary:   "app-darwin-arm64",
			platform: "darwin_amd64",
			want:     false,
		},
		{
			name:     "invalid platform format",
			binary:   "app-darwin-amd64",
			platform: "invalid",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsBinaryForPlatform(tt.binary, tt.platform); got != tt.want {
				t.Errorf("IsBinaryForPlatform() = %v, want %v", got, tt.want)
			}
		})
	}
}
