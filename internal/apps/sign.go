package apps

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"aead.dev/minisign"
)

// Key is the public key Kite's own index is signed with. kite-plus/apps
// publishes it as minisign.pub, and signs index.json with its private half
// into index.json.minisig; an index this key did not sign is refused.
const Key = "RWS7FFNcKsshXtrnjri11Qk9W6KWQxa1E+bPNr08Bm4vjNbjqyq+FEEt"

// ParseKey reads a minisign public key as a .pub file's second line has it.
func ParseKey(text string) (minisign.PublicKey, error) {
	var key minisign.PublicKey
	if err := key.UnmarshalText([]byte(strings.TrimSpace(text))); err != nil {
		return key, fmt.Errorf("%q is not a minisign public key", strings.TrimSpace(text))
	}
	return key, nil
}

// ErrUntrusted is reported, wrapped, when an index is not signed by the key
// the client trusts, or is older than one the client has already seen,
// which is how a copy kept from before a version was yanked would be
// passed off as current.
var ErrUntrusted = errors.New("untrusted")

type untrusted struct{ error }

func (untrusted) Is(target error) bool { return target == ErrUntrusted }
func (u untrusted) Unwrap() error      { return u.error }

// verify checks that sig is the key's signature of data.
func (c *Client) verify(data, sig []byte) error {
	if !minisign.Verify(c.Key, data, sig) {
		return untrusted{errors.New("it is not signed by the key Kite trusts for this index")}
	}
	return nil
}

// unsigned says why the signature of an index could not be read. An
// address that answers without one serves an index nobody vouches for; a
// network failure says nothing about the index.
func unsigned(err error) error {
	err = fmt.Errorf("its signature could not be read: %w", err)
	var status answered
	if errors.As(err, &status) {
		return untrusted{err}
	}
	return err
}

// older reports whether an index generated at generated is older than the
// newest one seen, both as the index writes them.
func older(generated, newest string) bool {
	g, err := time.Parse(time.RFC3339, generated)
	if err != nil {
		return false
	}
	n, err := time.Parse(time.RFC3339, newest)
	return err == nil && g.Before(n)
}
