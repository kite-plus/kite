// Package project locates a Kite project on disk and wires together the pieces
// that every command needs.
package project

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/kite-plus/kite/internal/config"
	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/store/file"
)

// ConfigName is the project configuration file that marks a project root.
const ConfigName = config.Name

// ErrNotFound is returned when no project root can be located.
var ErrNotFound = errors.New("project: no kite.yaml found in this directory or any parent")

// Project is an opened Kite project.
type Project struct {
	Root  string
	Types *content.Registry
}

// Open finds the project root by walking up from dir and returns it.
func Open(dir string) (*Project, error) {
	root, err := FindRoot(dir)
	if err != nil {
		return nil, err
	}
	return &Project{Root: root, Types: content.DefaultRegistry()}, nil
}

// FindRoot walks up from dir looking for the project marker file.
func FindRoot(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for {
		_, err := os.Stat(filepath.Join(abs, ConfigName))
		if err == nil {
			return abs, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return "", ErrNotFound
		}
		abs = parent
	}
}

// Scanner returns a scanner over the project's content.
func (p *Project) Scanner() *file.Scanner { return file.NewScanner(p.Root, p.Types) }

// Writer returns the project's content writer.
func (p *Project) Writer() *file.Writer { return file.NewWriter(p.Root, p.Types) }

// TypeOrError resolves a kind, reporting the available kinds when it is
// unknown.
func (p *Project) TypeOrError(kind string) (*content.Type, error) {
	if t := p.Types.Get(content.Kind(kind)); t != nil {
		return t, nil
	}
	return nil, fmt.Errorf("unknown content kind %q (known kinds: %v)", kind, p.Types.Kinds())
}
