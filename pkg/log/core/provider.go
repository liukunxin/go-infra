package core

import (
	"os"
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

// FileProvider (简单文件写)
type FileProvider struct {
	f *os.File
}

func NewFileProvider(path string) (*FileProvider, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	return &FileProvider{f: f}, nil
}

func (p *FileProvider) WriteLine(b []byte) {
	_, _ = p.f.Write(b)
}

func (p *FileProvider) Close() error {
	return p.f.Close()
}
