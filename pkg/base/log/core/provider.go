package core

import (
	"io"
	"os"

	"gopkg.in/lumberjack.v2"
)

type Provider interface {
	WriteLine(b []byte)
	Close() error
}

// StdProvider (可异步无锁写)
type StdProvider struct {
	out *os.File
}

func NewStdProvider() *StdProvider {
	return &StdProvider{out: os.Stdout}
}

func (p *StdProvider) WriteLine(b []byte) {
	_, _ = p.out.Write(b)
}

func (p *StdProvider) Close() error {
	return nil
}

// FileProvider writes to a file that is automatically rotated via lumberjack.
// MaxSizeMB  – rotate when the file reaches this size (default 100 MB).
// MaxBackups – number of old log files to keep (default 30).
// MaxAgeDays – maximum number of days to retain old log files (default 30).
// Compress   – gzip-compress rotated files.
type FileProvider struct {
	w io.WriteCloser
}

type FileProviderOptions struct {
	MaxSizeMB  int  // megabytes before rotation (default 100)
	MaxBackups int  // old files to keep (default 30)
	MaxAgeDays int  // days to retain (default 30)
	Compress   bool // gzip rotated files
}

func NewFileProvider(path string) (*FileProvider, error) {
	return NewFileProviderWithOptions(path, FileProviderOptions{})
}

func NewFileProviderWithOptions(path string, opts FileProviderOptions) (*FileProvider, error) {
	if opts.MaxSizeMB <= 0 {
		opts.MaxSizeMB = 100
	}
	if opts.MaxBackups <= 0 {
		opts.MaxBackups = 30
	}
	if opts.MaxAgeDays <= 0 {
		opts.MaxAgeDays = 30
	}
	// Probe writability with a plain open/close so we surface permission errors at
	// startup — without triggering a lumberjack rotation (Rotate() would rename the
	// current log file and create a new one, wasting a rotation slot on every restart).
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	_ = f.Close()

	return &FileProvider{w: &lumberjack.Logger{
		Filename:   path,
		MaxSize:    opts.MaxSizeMB,
		MaxBackups: opts.MaxBackups,
		MaxAge:     opts.MaxAgeDays,
		Compress:   opts.Compress,
		LocalTime:  true,
	}}, nil
}

func (p *FileProvider) WriteLine(b []byte) {
	_, _ = p.w.Write(b)
}

func (p *FileProvider) Close() error {
	return p.w.Close()
}
