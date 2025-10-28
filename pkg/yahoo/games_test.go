package yahoo

import (
	"testing"
)

func TestGetGameID(t *testing.T) {
	tests := []struct {
		name      string
		gameCode  string
		season    int
		want      int
		wantError bool
	}{
		{
			name:     "MLB 2024",
			gameCode: "mlb",
			season:   2024,
			want:     431,
		},
		{
			name:     "NFL 2024",
			gameCode: "nfl",
			season:   2024,
			want:     449,
		},
		{
			name:     "NBA 2024",
			gameCode: "nba",
			season:   2024,
			want:     454,
		},
		{
			name:     "NHL 2024",
			gameCode: "nhl",
			season:   2024,
			want:     453,
		},
		{
			name:      "Invalid game code",
			gameCode:  "soccer",
			season:    2024,
			wantError: true,
		},
		{
			name:      "Invalid season",
			gameCode:  "nfl",
			season:    1999,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetGameID(tt.gameCode, tt.season)
			if tt.wantError {
				if err == nil {
					t.Errorf("GetGameID() expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("GetGameID() unexpected error: %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("GetGameID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetGameKey(t *testing.T) {
	tests := []struct {
		name      string
		gameCode  string
		season    int
		want      string
		wantError bool
	}{
		{
			name:     "MLB 2024",
			gameCode: "mlb",
			season:   2024,
			want:     "431",
		},
		{
			name:     "NFL 2023",
			gameCode: "nfl",
			season:   2023,
			want:     "423",
		},
		{
			name:      "Invalid game",
			gameCode:  "invalid",
			season:    2024,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetGameKey(tt.gameCode, tt.season)
			if tt.wantError {
				if err == nil {
					t.Errorf("GetGameKey() expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("GetGameKey() unexpected error: %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("GetGameKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
