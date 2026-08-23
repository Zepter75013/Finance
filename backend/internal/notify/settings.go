package notify

import (
	"encoding/json"
	"os"
)

// stateFile mémorise la dernière date d'envoi du résumé quotidien, pour ne
// jamais en envoyer deux le même jour civil — même pattern que
// backup-settings.json.
const stateFile = "notify-state.json"

type persistedState struct {
	LastDigestDate string `json:"last_digest_date"`
}

func loadLastDigestDate() string {
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return ""
	}

	var s persistedState
	if err := json.Unmarshal(data, &s); err != nil {
		return ""
	}

	return s.LastDigestDate
}

func saveLastDigestDate(date string) error {
	data, err := json.MarshalIndent(persistedState{LastDigestDate: date}, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(stateFile, data, 0o644)
}
