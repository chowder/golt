package golt

import (
	"os"
	"os/exec"
)

func LaunchGame(sessionId string, account *Account) error {
	game, ok := os.LookupEnv("GOLT_GAME_PATH")
	if !ok {
		game = "RuneLite.AppImage"
	}

	cmd := exec.Command(game)

	cmd.Env = append(os.Environ(), []string{
		"JX_SESSION_ID=" + sessionId,
		"JX_CHARACTER_ID=" + account.AccountId,
		"JX_DISPLAY_NAME=" + account.DisplayName,
	}...)

	return cmd.Start()
}
