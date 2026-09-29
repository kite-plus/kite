package markdown

import (
	"fmt"
	"strconv"
	"strings"
)

// tag is one shortcode tag as a body writes it: {{< name params >}},
// {{< /name >}} or {{< name params />}}, or the same between {{% and %}}.
type tag struct {
	start, end int // the tag's bytes in the body
	line       int

	name       string
	closing    bool // {{< /name >}}
	selfClosed bool // {{< name />}}, which takes no closing tag

	positional []any
	named      map[string]any
}

// escaped reports whether a tag written at s[at:] is commented out, as
// {{</* name */>}}, and so stands for the text of a tag rather than a call.
// It returns the text it stands for and where it ends.
func escaped(s string, at int) (literal string, end int, ok bool) {
	open, shut := s[at:at+3], ">}}"
	if open == "{{%" {
		shut = "%}}"
	}
	if !strings.HasPrefix(s[at+3:], "/*") {
		return "", 0, false
	}
	stop := strings.Index(s[at+5:], "*/"+shut)
	if stop < 0 {
		return "", 0, false
	}
	inside := s[at+5 : at+5+stop]
	return open + inside + shut, at + 5 + stop + 2 + len(shut), true
}

// lexTag reads the tag written at s[at:], which starts with {{< or {{%. Its
// syntax is Hugo's, so that content moved from a Hugo site reads the same:
// parameters are separated by spaces and given in order or by name, never
// both, and a value is "quoted", `raw` or a single word. A word that reads as
// true, false or a number is that value; anything quoted stays text.
func lexTag(s string, at, line int) (tag, error) {
	t := tag{start: at, line: line}
	end := ">}}"
	if s[at+2] == '%' {
		end = "%}}"
	}
	fail := func(format string, args ...any) (tag, error) {
		return tag{}, &ShortcodeError{Line: line, Name: t.name, Err: fmt.Errorf(format, args...)}
	}

	i := skipSpace(s, at+3)
	if i < len(s) && s[i] == '/' {
		t.closing = true
		i = skipSpace(s, i+1)
	}
	name, i := lexName(s, i)
	if name == "" {
		return fail("a tag needs a name of letters, digits, - and _, as %s", "{{< figure >}}")
	}
	t.name = name
	if i < len(s) && !isSpace(s[i]) && !strings.HasPrefix(s[i:], end) && !strings.HasPrefix(s[i:], "/"+end) {
		return fail("a name is letters, digits, - and _, separated by /")
	}

	for {
		i = skipSpace(s, i)
		switch {
		case strings.HasPrefix(s[i:], end):
			t.end = i + len(end)
			return t, nil
		case strings.HasPrefix(s[i:], "/"+end):
			if t.closing {
				return fail("a closing tag cannot also close itself")
			}
			t.selfClosed = true
			t.end = i + 1 + len(end)
			return t, nil
		case i >= len(s):
			return fail("the tag is never closed with %s", end)
		case t.closing:
			return fail("a closing tag takes no parameters")
		}

		key, value, next, err := lexParam(s, i, end)
		if err != nil {
			return fail("%v", err)
		}
		i = next
		if key == "" {
			if t.named != nil {
				return fail("parameters are given by name or in order, not both")
			}
			t.positional = append(t.positional, value)
			continue
		}
		if t.positional != nil {
			return fail("parameters are given by name or in order, not both")
		}
		if _, dup := t.named[key]; dup {
			return fail("%s is given twice", key)
		}
		if t.named == nil {
			t.named = make(map[string]any)
		}
		t.named[key] = value
	}
}

// lexName reads a shortcode's name: words of letters, digits, - and _, which
// a / may join to name a template in a folder of its own.
func lexName(s string, i int) (string, int) {
	start := i
	for {
		j := i
		for j < len(s) && isNameByte(s[j]) {
			j++
		}
		if j == i {
			// A / not followed by a word belongs to what comes after the
			// name, as in {{< name/>}}.
			if i > start && s[i-1] == '/' {
				i--
			}
			return s[start:i], i
		}
		i = j
		if i >= len(s) || s[i] != '/' {
			return s[start:i], i
		}
		i++
	}
}

// lexParam reads one parameter at s[i:], which is not space: key=value, or a
// value alone.
func lexParam(s string, i int, end string) (key string, value any, next int, err error) {
	j := i
	for j < len(s) && isNameByte(s[j]) {
		j++
	}
	if j > i && j < len(s) && s[j] == '=' {
		key, i = s[i:j], j+1
		if i >= len(s) || isSpace(s[i]) || strings.HasPrefix(s[i:], end) {
			return "", nil, 0, fmt.Errorf("%s= has no value", key)
		}
	}
	value, next, err = lexValue(s, i, end)
	return key, value, next, err
}

// lexValue reads one value at s[i:]: a quoted string, in which \" is a quote,
// a raw string between backquotes, or a word, which ends at a space or at the
// end of the tag.
func lexValue(s string, i int, end string) (any, int, error) {
	switch s[i] {
	case '"':
		var b strings.Builder
		for j := i + 1; j < len(s); j++ {
			switch {
			case s[j] == '\\' && j+1 < len(s) && s[j+1] == '"':
				b.WriteByte('"')
				j++
			case s[j] == '"':
				return b.String(), j + 1, nil
			default:
				b.WriteByte(s[j])
			}
		}
		return nil, 0, fmt.Errorf("a quoted value is never closed")
	case '`':
		stop := strings.IndexByte(s[i+1:], '`')
		if stop < 0 {
			return nil, 0, fmt.Errorf("a value in backquotes is never closed")
		}
		return s[i+1 : i+1+stop], i + 1 + stop + 1, nil
	}
	j := i
	for j < len(s) && !isSpace(s[j]) && !strings.HasPrefix(s[j:], end) && !strings.HasPrefix(s[j:], "/"+end) {
		j++
	}
	return typed(s[i:j]), j, nil
}

// typed reads a word as the value it spells: true and false, an integer or a
// decimal number. Any other word is text.
func typed(word string) any {
	switch word {
	case "true":
		return true
	case "false":
		return false
	}
	if strings.Trim(word, "+-0123456789.eE") != "" || !strings.ContainsAny(word, "0123456789") {
		return word
	}
	if n, err := strconv.Atoi(word); err == nil {
		return n
	}
	if f, err := strconv.ParseFloat(word, 64); err == nil {
		return f
	}
	return word
}

func isNameByte(c byte) bool {
	return c == '-' || c == '_' || '0' <= c && c <= '9' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z'
}

func isSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }

func skipSpace(s string, i int) int {
	for i < len(s) && isSpace(s[i]) {
		i++
	}
	return i
}
