package backup

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"finance/backend/internal/config"
)

var ErrInvalidName = errors.New("invalid backup filename")
var ErrDialogCancelled = errors.New("l'utilisateur a annulé la sélection")

const fileSuffix = ".sql"

// autoBackupPrefix distingue les sauvegardes créées par le planificateur de
// fond des sauvegardes manuelles (même nommage sinon) — seules celles-ci
// sont candidates à la purge par rétention.
const autoBackupPrefix = "finance-backup-auto-"

type Service struct {
	cfg config.Config
	mu  sync.RWMutex

	dir               string
	autoBackupEnabled bool
	retentionDays     int
	lastAutoBackupAt  int64
}

func NewService(cfg config.Config, defaultDir string) (*Service, error) {
	dir := defaultDir
	autoBackupEnabled := true
	retentionDays := defaultRetentionDays
	var lastAutoBackupAt int64

	if persisted, ok := loadPersistedSettings(); ok {
		dir = persisted.Directory
		autoBackupEnabled = persisted.AutoBackupEnabled
		retentionDays = persisted.RetentionDays
		lastAutoBackupAt = persisted.LastAutoBackupAt
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(absDir, 0o755); err != nil {
		return nil, err
	}

	return &Service{
		cfg:               cfg,
		dir:               absDir,
		autoBackupEnabled: autoBackupEnabled,
		retentionDays:     retentionDays,
		lastAutoBackupAt:  lastAutoBackupAt,
	}, nil
}

// persistedSnapshot copie l'état courant sous verrou, pour l'écrire dans
// backup-settings.json sans jamais écraser un champ que l'appelant ne
// modifie pas (le fichier est réécrit en entier à chaque sauvegarde).
func (s *Service) persistedSnapshot() persistedSettings {
	return persistedSettings{
		Directory:         s.dir,
		AutoBackupEnabled: s.autoBackupEnabled,
		RetentionDays:     s.retentionDays,
		LastAutoBackupAt:  s.lastAutoBackupAt,
	}
}

// Directory returns the folder backups are currently read from/written to.
func (s *Service) Directory() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.dir
}

// SetDirectory switches where future backups are stored. It validates that
// the path is usable (creatable and writable) before persisting it, and
// does NOT move backups already sitting in the previous directory.
func (s *Service) SetDirectory(dir string) error {
	trimmed := strings.TrimSpace(dir)
	if trimmed == "" {
		return errors.New("le dossier ne peut pas être vide")
	}

	if !filepath.IsAbs(trimmed) {
		return errors.New("indique un chemin absolu (ex: /Users/toi/Sauvegardes)")
	}

	if err := os.MkdirAll(trimmed, 0o755); err != nil {
		return fmt.Errorf("impossible de créer ce dossier: %w", err)
	}

	probe := filepath.Join(trimmed, ".write-test")
	if err := os.WriteFile(probe, []byte("ok"), 0o644); err != nil {
		return fmt.Errorf("ce dossier n'est pas accessible en écriture: %w", err)
	}
	os.Remove(probe)

	s.mu.RLock()
	snapshot := s.persistedSnapshot()
	s.mu.RUnlock()
	snapshot.Directory = trimmed

	if err := savePersistedSettings(snapshot); err != nil {
		return fmt.Errorf("échec de l'enregistrement du réglage: %w", err)
	}

	s.mu.Lock()
	s.dir = trimmed
	s.mu.Unlock()

	return nil
}

// AutoBackupEnabled indique si la sauvegarde automatique quotidienne est activée.
func (s *Service) AutoBackupEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.autoBackupEnabled
}

// RetentionDays indique combien de jours les sauvegardes automatiques sont conservées.
func (s *Service) RetentionDays() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.retentionDays
}

// SetAutoBackupSettings active/désactive la sauvegarde automatique et fixe sa rétention.
func (s *Service) SetAutoBackupSettings(enabled bool, retentionDays int) error {
	if retentionDays <= 0 {
		return errors.New("la rétention doit être d'au moins 1 jour")
	}

	s.mu.RLock()
	snapshot := s.persistedSnapshot()
	s.mu.RUnlock()
	snapshot.AutoBackupEnabled = enabled
	snapshot.RetentionDays = retentionDays

	if err := savePersistedSettings(snapshot); err != nil {
		return fmt.Errorf("échec de l'enregistrement du réglage: %w", err)
	}

	s.mu.Lock()
	s.autoBackupEnabled = enabled
	s.retentionDays = retentionDays
	s.mu.Unlock()

	return nil
}

// PickDirectoryDialog opens the native macOS "choose folder" dialog on the
// machine running this server (it runs on the user's own desktop, so this is
// a real Finder picker, not something shown inside the browser) and returns
// the POSIX path the user selected.
func (s *Service) PickDirectoryDialog(ctx context.Context) (string, error) {
	if runtime.GOOS != "darwin" {
		return "", errors.New("le sélecteur de dossier natif n'est disponible que sur macOS")
	}

	script := `POSIX path of (choose folder with prompt "Choisis le dossier de sauvegarde de Finance UI")`
	cmd := exec.CommandContext(ctx, "osascript", "-e", script)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if strings.Contains(stderr.String(), "User canceled") || strings.Contains(stderr.String(), "-128") {
			return "", ErrDialogCancelled
		}

		return "", fmt.Errorf("osascript: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	path := strings.TrimSpace(stdout.String())

	return strings.TrimSuffix(path, string(filepath.Separator)), nil
}

func (s *Service) List() ([]BackupFile, error) {
	entries, err := os.ReadDir(s.Directory())
	if err != nil {
		return nil, err
	}

	files := make([]BackupFile, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), fileSuffix) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		files = append(files, BackupFile{
			Name:      entry.Name(),
			SizeBytes: info.Size(),
			CreatedAt: info.ModTime(),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].CreatedAt.After(files[j].CreatedAt)
	})

	return files, nil
}

// resolvePath validates a client-supplied backup filename and returns its
// absolute path on disk. It rejects anything that isn't a bare ".sql"
// filename (no directory separators, no "..") to prevent path traversal.
func (s *Service) resolvePath(name string) (string, error) {
	clean := filepath.Base(name)
	if clean != name || clean == "." || clean == string(filepath.Separator) || !strings.HasSuffix(clean, fileSuffix) {
		return "", ErrInvalidName
	}

	return filepath.Join(s.Directory(), clean), nil
}

func (s *Service) dumpArgs() []string {
	return []string{
		"-h", s.cfg.DBHost,
		"-P", s.cfg.DBPort,
		"-u", s.cfg.DBUser,
		"--single-transaction",
		"--routines",
		"--events",
		s.cfg.DBName,
	}
}

func (s *Service) restoreArgs() []string {
	return []string{
		"-h", s.cfg.DBHost,
		"-P", s.cfg.DBPort,
		"-u", s.cfg.DBUser,
		s.cfg.DBName,
	}
}

func (s *Service) withPassword(cmd *exec.Cmd) {
	if s.cfg.DBPassword != "" {
		cmd.Env = append(os.Environ(), "MYSQL_PWD="+s.cfg.DBPassword)
	}
}

// Create dumps the full database to a new timestamped file and returns its metadata.
func (s *Service) Create(ctx context.Context) (BackupFile, error) {
	return s.dumpTo(ctx, fmt.Sprintf("finance-backup-%s%s", time.Now().Format("20060102-150405"), fileSuffix))
}

// CreateAuto est l'équivalent de Create pour le planificateur de fond — seul
// le préfixe du nom de fichier diffère, afin que la purge par rétention
// puisse cibler uniquement les sauvegardes automatiques.
func (s *Service) CreateAuto(ctx context.Context) (BackupFile, error) {
	return s.dumpTo(ctx, fmt.Sprintf("%s%s%s", autoBackupPrefix, time.Now().Format("20060102-150405"), fileSuffix))
}

// RunScheduledBackupIfDue exécute une sauvegarde automatique si elle est
// activée et que la précédente date de plus de 24h, puis purge les
// sauvegardes automatiques plus vieilles que la rétention configurée — la
// purge tourne à CHAQUE appel (chaque heure), pas seulement lorsqu'une
// nouvelle sauvegarde vient d'être créée, pour que la rétention reste
// respectée en continu même si la création du jour a déjà eu lieu.
// Un échec de création est simplement journalisé par l'appelant et retenté
// au prochain passage horaire, sans mécanisme de retry séparé.
func (s *Service) RunScheduledBackupIfDue(ctx context.Context) error {
	s.mu.RLock()
	enabled := s.autoBackupEnabled
	lastAutoBackupAt := s.lastAutoBackupAt
	retentionDays := s.retentionDays
	s.mu.RUnlock()

	if !enabled {
		return nil
	}

	if lastAutoBackupAt == 0 || time.Since(time.Unix(lastAutoBackupAt, 0)) >= 24*time.Hour {
		if _, err := s.CreateAuto(ctx); err != nil {
			return err
		}

		s.mu.Lock()
		s.lastAutoBackupAt = time.Now().Unix()
		snapshot := s.persistedSnapshot()
		s.mu.Unlock()

		if err := savePersistedSettings(snapshot); err != nil {
			return fmt.Errorf("échec de l'enregistrement du réglage: %w", err)
		}
	}

	return s.pruneAutoBackups(retentionDays)
}

// pruneAutoBackups supprime les sauvegardes automatiques dont la date de
// modification dépasse la rétention configurée — jamais les sauvegardes
// manuelles ni les filets de sécurité pre-restore/upload.
func (s *Service) pruneAutoBackups(retentionDays int) error {
	entries, err := os.ReadDir(s.Directory())
	if err != nil {
		return err
	}

	cutoff := time.Duration(retentionDays) * 24 * time.Hour

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), autoBackupPrefix) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if time.Since(info.ModTime()) > cutoff {
			_ = os.Remove(filepath.Join(s.Directory(), entry.Name()))
		}
	}

	return nil
}

func (s *Service) dumpTo(ctx context.Context, filename string) (BackupFile, error) {
	path := filepath.Join(s.Directory(), filename)

	outFile, err := os.Create(path)
	if err != nil {
		return BackupFile{}, err
	}
	defer outFile.Close()

	cmd := exec.CommandContext(ctx, "mysqldump", s.dumpArgs()...)
	s.withPassword(cmd)
	cmd.Stdout = outFile

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		outFile.Close()
		os.Remove(path)

		return BackupFile{}, fmt.Errorf("mysqldump: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	info, err := outFile.Stat()
	if err != nil {
		return BackupFile{}, err
	}

	return BackupFile{Name: filename, SizeBytes: info.Size(), CreatedAt: info.ModTime()}, nil
}

func (s *Service) OpenForDownload(name string) (*os.File, BackupFile, error) {
	path, err := s.resolvePath(name)
	if err != nil {
		return nil, BackupFile{}, err
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, BackupFile{}, err
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, BackupFile{}, err
	}

	return file, BackupFile{Name: name, SizeBytes: info.Size(), CreatedAt: info.ModTime()}, nil
}

func (s *Service) Delete(name string) error {
	path, err := s.resolvePath(name)
	if err != nil {
		return err
	}

	return os.Remove(path)
}

// restoreFile pipes the given SQL file into the `mysql` client, replacing the
// current database content. As a safety net it always creates a fresh
// "pre-restore" backup of the current state first, so an accidental or wrong
// restore is never truly unrecoverable.
func (s *Service) restoreFile(ctx context.Context, path string) error {
	if _, err := s.dumpTo(ctx, fmt.Sprintf("finance-backup-pre-restore-%s%s", time.Now().Format("20060102-150405"), fileSuffix)); err != nil {
		return fmt.Errorf("échec de la sauvegarde de sécurité avant restauration: %w", err)
	}

	inFile, err := os.Open(path)
	if err != nil {
		return err
	}
	defer inFile.Close()

	cmd := exec.CommandContext(ctx, "mysql", s.restoreArgs()...)
	s.withPassword(cmd)
	cmd.Stdin = inFile

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mysql: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	return nil
}

// RestoreExisting restores the database from one of the backups already
// stored on disk.
func (s *Service) RestoreExisting(ctx context.Context, name string) error {
	path, err := s.resolvePath(name)
	if err != nil {
		return err
	}

	return s.restoreFile(ctx, path)
}

// RestoreUpload saves an uploaded SQL dump alongside the other backups (so it
// shows up in the list too) and restores the database from it.
func (s *Service) RestoreUpload(ctx context.Context, content io.Reader) (BackupFile, error) {
	filename := fmt.Sprintf("finance-backup-upload-%s%s", time.Now().Format("20060102-150405"), fileSuffix)
	path := filepath.Join(s.Directory(), filename)

	outFile, err := os.Create(path)
	if err != nil {
		return BackupFile{}, err
	}

	if _, err := io.Copy(outFile, content); err != nil {
		outFile.Close()
		os.Remove(path)

		return BackupFile{}, err
	}

	info, err := outFile.Stat()
	outFile.Close()
	if err != nil {
		os.Remove(path)
		return BackupFile{}, err
	}

	if err := s.restoreFile(ctx, path); err != nil {
		return BackupFile{}, err
	}

	return BackupFile{Name: filename, SizeBytes: info.Size(), CreatedAt: info.ModTime()}, nil
}
