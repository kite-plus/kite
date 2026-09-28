// Package metajson writes an item's front matter values as JSON, the way the
// index keeps them, and reads them back as the Go types they were.
//
// Plain JSON hands back a string for every date and a float64 for every
// number, so a template could neither format a date a list shows nor compare
// a count with the 8 it writes. A time is written as {"$time": "..."}, and a
// float with no fraction keeps a ".0" that tells it from an integer.
package metajson

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
)

// timeKey names the one key of the object a time is written as.
const timeKey = "$time"

// Marshal writes meta as JSON, keys in order.
func Marshal(meta map[string]any) ([]byte, error) {
	var b bytes.Buffer
	if meta == nil {
		meta = map[string]any{}
	}
	if err := write(&b, meta); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func write(b *bytes.Buffer, v any) error {
	switch x := v.(type) {
	case time.Time:
		b.WriteString(`{"` + timeKey + `":`)
		b.WriteString(strconv.Quote(x.Format(time.RFC3339Nano)))
		b.WriteByte('}')
	case float64:
		return writeFloat(b, x)
	case float32:
		return writeFloat(b, float64(x))
	case map[string]any:
		b.WriteByte('{')
		for i, k := range slices.Sorted(maps.Keys(x)) {
			if i > 0 {
				b.WriteByte(',')
			}
			key, err := json.Marshal(k)
			if err != nil {
				return err
			}
			b.Write(key)
			b.WriteByte(':')
			if err := write(b, x[k]); err != nil {
				return err
			}
		}
		b.WriteByte('}')
	case []any:
		b.WriteByte('[')
		for i, item := range x {
			if i > 0 {
				b.WriteByte(',')
			}
			if err := write(b, item); err != nil {
				return err
			}
		}
		b.WriteByte(']')
	default:
		out, err := json.Marshal(x)
		if err != nil {
			return err
		}
		b.Write(out)
	}
	return nil
}

func writeFloat(b *bytes.Buffer, f float64) error {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return fmt.Errorf("metajson: %v cannot be written as JSON", f)
	}
	out, err := json.Marshal(f)
	if err != nil {
		return err
	}
	b.Write(out)
	if !bytes.ContainsAny(out, ".eE") {
		b.WriteString(".0")
	}
	return nil
}

// Unmarshal reads what Marshal wrote.
func Unmarshal(data []byte) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var meta map[string]any
	if err := dec.Decode(&meta); err != nil {
		return nil, err
	}
	for k, v := range meta {
		meta[k] = restore(v)
	}
	return meta, nil
}

func restore(v any) any {
	switch x := v.(type) {
	case json.Number:
		s := x.String()
		if !strings.ContainsAny(s, ".eE") {
			if i, err := strconv.ParseInt(s, 10, 0); err == nil {
				return int(i)
			}
		}
		f, _ := x.Float64()
		return f
	case map[string]any:
		if s, ok := x[timeKey].(string); ok && len(x) == 1 {
			if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
				return t
			}
		}
		for k, item := range x {
			x[k] = restore(item)
		}
		return x
	case []any:
		for i, item := range x {
			x[i] = restore(item)
		}
		return x
	}
	return v
}
