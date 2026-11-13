package img

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
	"time"
)

// version names the code that makes pictures. It is part of every key, so a
// change to how a picture comes out makes it again instead of handing out
// what an older build made.
const version = "1"

// Made is a picture a recipe made.
type Made struct {
	Key           string
	Format        string
	Width, Height int

	data []byte // kept when there is no directory to keep it in
	file string
}

// Ext is the extension of the file a picture is written to.
func (m *Made) Ext() string { return extension(m.Format) }

// MediaType is the picture's media type.
func (m *Made) MediaType() string { return "image/" + m.Format }

// Bytes are the picture's bytes.
func (m *Made) Bytes() ([]byte, error) {
	if m.data != nil {
		return m.data, nil
	}
	return os.ReadFile(m.file)
}

// Processor makes pictures by recipe and keeps each it made: on disk when it
// has a directory, which a later build or server finds them in, and in memory
// otherwise.
type Processor struct {
	dir string

	mu      sync.Mutex
	made    map[string]*Made
	making  map[string]chan struct{}
	sources map[string]source
	busy    chan struct{}
}

type source struct {
	size int64
	mod  time.Time
	hash string
}

// NewProcessor returns a processor keeping what it makes in dir, or in
// memory when dir is empty.
func NewProcessor(dir string) *Processor {
	// Each picture being made holds a few copies of itself decoded, so only
	// a few are made at once.
	busy := max(1, min(4, runtime.GOMAXPROCS(0)/2))
	return &Processor{
		dir:     dir,
		made:    make(map[string]*Made),
		making:  make(map[string]chan struct{}),
		sources: make(map[string]source),
		busy:    make(chan struct{}, busy),
	}
}

// Hash names a source picture by its bytes. The name is kept by path, size
// and time of change, so a picture is read again only once it changes.
func (p *Processor) Hash(path string, size int64, mod time.Time, read func() ([]byte, error)) (string, error) {
	p.mu.Lock()
	s, ok := p.sources[path]
	p.mu.Unlock()
	if ok && s.size == size && s.mod.Equal(mod) {
		return s.hash, nil
	}
	data, err := read()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])
	p.mu.Lock()
	p.sources[path] = source{size: size, mod: mod, hash: hash}
	p.mu.Unlock()
	return hash, nil
}

// Key names what a recipe makes of a source, by the source's hash, the
// recipe and the version of the code that makes it.
func Key(sourceHash string, r Recipe) string {
	sum := sha256.Sum256([]byte(version + "\n" + sourceHash + "\n" + r.String()))
	return hex.EncodeToString(sum[:8])
}

// Make returns what a recipe makes of a source, made now or found among the
// pictures made before. read gives the source's bytes.
func (p *Processor) Make(sourceHash string, read func() ([]byte, error), r Recipe) (*Made, error) {
	key := Key(sourceHash, r)
	for {
		p.mu.Lock()
		if m, ok := p.made[key]; ok {
			p.mu.Unlock()
			return m, nil
		}
		wait, running := p.making[key]
		if !running {
			done := make(chan struct{})
			p.making[key] = done
			p.mu.Unlock()
			m, err := p.makeOne(key, read, r)
			p.mu.Lock()
			if err == nil {
				p.made[key] = m
			}
			delete(p.making, key)
			p.mu.Unlock()
			close(done)
			return m, err
		}
		p.mu.Unlock()
		<-wait
	}
}

func (p *Processor) makeOne(key string, read func() ([]byte, error), r Recipe) (*Made, error) {
	if m, ok := p.stored(key); ok {
		return m, nil
	}
	p.busy <- struct{}{}
	defer func() { <-p.busy }()

	data, err := read()
	if err != nil {
		return nil, err
	}
	out, info, err := produce(data, r)
	if err != nil {
		return nil, err
	}
	m := &Made{Key: key, Format: info.Format, Width: info.Width, Height: info.Height}
	if p.dir == "" {
		m.data = out
		return m, nil
	}
	m.file = filepath.Join(p.dir, key+"."+m.Ext())
	if err := writeFile(m.file, out); err != nil {
		// A cache that cannot be written leaves the picture in memory.
		m.data, m.file = out, ""
	}
	return m, nil
}

// stored finds a picture an earlier run made, on disk.
func (p *Processor) stored(key string) (*Made, bool) {
	if p.dir == "" {
		return nil, false
	}
	for _, format := range []string{"jpeg", "png", "gif", "webp"} {
		file := filepath.Join(p.dir, key+"."+extension(format))
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		info, err := Inspect(data)
		if err != nil {
			continue
		}
		return &Made{Key: key, Format: info.Format, Width: info.Width, Height: info.Height, file: file}, true
	}
	return nil, false
}

// madeName is how the file a made picture is published as ends: an
// underscore, its key and its extension.
var madeName = regexp.MustCompile(`_([0-9a-f]{16})\.(jpg|png|gif|webp)$`)

// Lookup finds the picture a published file name holds, among the pictures
// made before, as a server answers a request for one.
func (p *Processor) Lookup(name string) (*Made, bool) {
	m := madeName.FindStringSubmatch(name)
	if m == nil {
		return nil, false
	}
	p.mu.Lock()
	made, ok := p.made[m[1]]
	p.mu.Unlock()
	if ok {
		return made, made.Ext() == m[2]
	}
	made, ok = p.stored(m[1])
	if !ok || made.Ext() != m[2] {
		return nil, false
	}
	p.mu.Lock()
	p.made[made.Key] = made
	p.mu.Unlock()
	return made, true
}

// PublishedName is the name a made picture is published under, beside the
// source it was made from: river.jpg becomes river_<key>.jpg.
func PublishedName(base string, m *Made) string {
	return fmt.Sprintf("%s_%s.%s", base, m.Key, m.Ext())
}

func extension(format string) string {
	if format == "jpeg" {
		return "jpg"
	}
	return format
}

// writeFile writes a file whole or not at all, so a picture cut short by a
// crash is never found and handed out.
func writeFile(name string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(name), ".tmp-*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), name)
}
