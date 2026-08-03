package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatConsoleIdentity(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		serverName string
		expected   string
	}{
		{
			name:       "pterodactyl entrypoint identity",
			input:      "container@pterodactyl~ java -Xms128M -Xmx1024M",
			serverName: "Survie",
			expected:   "Photon-Daemon@Survie java -Xms128M -Xmx1024M",
		},
		{
			name:       "alternate identity",
			input:      "pterodactyl@container: Starting server",
			serverName: "Paper 1.21",
			expected:   "Photon-Daemon@Paper 1.21 Starting server",
		},
		{
			name:       "unrelated output",
			input:      "Done (4.201s)! For help, type help",
			serverName: "Survie",
			expected:   "Done (4.201s)! For help, type help",
		},
		{
			name:       "missing server name",
			input:      "container@pterodactyl~ ./start.sh",
			serverName: "",
			expected:   "container@pterodactyl~ ./start.sh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(formatConsoleIdentity([]byte(tt.input), tt.serverName)))
		})
	}
}
