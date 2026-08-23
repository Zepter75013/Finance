package backup

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeFileWithAge(t *testing.T, dir, name string, age time.Duration) {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("-- fake backup"), 0o644); err != nil {
		t.Fatalf("écriture de %s: %v", name, err)
	}

	mtime := time.Now().Add(-age)
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatalf("chtimes de %s: %v", name, err)
	}
}

func TestPruneAutoBackups(t *testing.T) {
	dir := t.TempDir()

	writeFileWithAge(t, dir, "finance-backup-auto-20260101-000000.sql", 40*24*time.Hour) // vieille auto -> supprimée
	writeFileWithAge(t, dir, "finance-backup-auto-20260820-000000.sql", 2*24*time.Hour)  // récente auto -> conservée
	writeFileWithAge(t, dir, "finance-backup-20260101-000000.sql", 40*24*time.Hour)      // vieille MANUELLE -> jamais supprimée
	writeFileWithAge(t, dir, "finance-backup-pre-restore-20260101-000000.sql", 40*24*time.Hour)
	writeFileWithAge(t, dir, "finance-backup-upload-20260101-000000.sql", 40*24*time.Hour)

	s := &Service{dir: dir, retentionDays: 30}

	if err := s.pruneAutoBackups(s.retentionDays); err != nil {
		t.Fatalf("pruneAutoBackups: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	remaining := make(map[string]bool)
	for _, entry := range entries {
		remaining[entry.Name()] = true
	}

	if remaining["finance-backup-auto-20260101-000000.sql"] {
		t.Error("la vieille sauvegarde automatique aurait dû être supprimée")
	}

	expectedKept := []string{
		"finance-backup-auto-20260820-000000.sql",
		"finance-backup-20260101-000000.sql",
		"finance-backup-pre-restore-20260101-000000.sql",
		"finance-backup-upload-20260101-000000.sql",
	}

	for _, name := range expectedKept {
		if !remaining[name] {
			t.Errorf("%s aurait dû être conservée", name)
		}
	}
}
