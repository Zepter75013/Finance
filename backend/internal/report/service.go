package report

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Service stores exactly one generated report (the most recent one) on
// disk: the PDF itself plus a small JSON sidecar with its metadata. No
// database table needed — same lightweight approach as the DB backups.
type Service struct {
	dir string
}

type Metadata struct {
	GeneratedAt time.Time `json:"generated_at"`
	Criteria    string    `json:"criteria"`
	SizeBytes   int64     `json:"size_bytes"`
}

const pdfFilename = "latest.pdf"
const metadataFilename = "latest.json"

var ErrNoReport = errors.New("no report generated yet")

func NewService(dir string) (*Service, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	return &Service{dir: dir}, nil
}

func (s *Service) pdfPath() string {
	return filepath.Join(s.dir, pdfFilename)
}

func (s *Service) metadataPath() string {
	return filepath.Join(s.dir, metadataFilename)
}

// SaveLatest overwrites the previously generated report (if any) with a new
// one — only the most recent report is ever kept.
func (s *Service) SaveLatest(criteria string, content io.Reader) (Metadata, error) {
	pdfPath := s.pdfPath()

	file, err := os.Create(pdfPath)
	if err != nil {
		return Metadata{}, err
	}

	size, err := io.Copy(file, content)
	if err != nil {
		file.Close()
		os.Remove(pdfPath)
		return Metadata{}, err
	}
	file.Close()

	meta := Metadata{
		GeneratedAt: time.Now(),
		Criteria:    criteria,
		SizeBytes:   size,
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return Metadata{}, err
	}

	if err := os.WriteFile(s.metadataPath(), data, 0o644); err != nil {
		return Metadata{}, err
	}

	return meta, nil
}

func (s *Service) GetLatestMetadata() (Metadata, error) {
	data, err := os.ReadFile(s.metadataPath())
	if err != nil {
		if os.IsNotExist(err) {
			return Metadata{}, ErrNoReport
		}
		return Metadata{}, err
	}

	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return Metadata{}, err
	}

	return meta, nil
}

func (s *Service) OpenLatestPdf() (*os.File, error) {
	file, err := os.Open(s.pdfPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoReport
		}
		return nil, err
	}

	return file, nil
}
