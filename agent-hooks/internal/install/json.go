package install

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// object is a JSON object that remembers its key order, so rewriting a
// hand-edited config only changes the entries this tool owns.
type object struct {
	keys []string
	vals map[string]any // *object, []any, string, json.Number, bool, or nil
}

func newObject() *object { return &object{vals: map[string]any{}} }

func (o *object) get(key string) (any, bool) {
	v, ok := o.vals[key]
	return v, ok
}

func (o *object) set(key string, v any) {
	if _, ok := o.vals[key]; !ok {
		o.keys = append(o.keys, key)
	}
	o.vals[key] = v
}

func (o *object) del(key string) {
	if _, ok := o.vals[key]; !ok {
		return
	}
	delete(o.vals, key)
	for i, k := range o.keys {
		if k == key {
			o.keys = append(o.keys[:i], o.keys[i+1:]...)
			break
		}
	}
}

func parseObject(data []byte) (*object, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	v, err := parseValue(dec)
	if err != nil {
		return nil, err
	}
	if rest := bytes.TrimSpace(data[dec.InputOffset():]); len(rest) > 0 {
		return nil, errors.New("trailing data after the JSON object")
	}
	o, ok := v.(*object)
	if !ok {
		return nil, errors.New("not a JSON object")
	}
	return o, nil
}

func parseValue(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			o := newObject()
			for dec.More() {
				k, err := dec.Token()
				if err != nil {
					return nil, err
				}
				v, err := parseValue(dec)
				if err != nil {
					return nil, err
				}
				o.set(k.(string), v)
			}
			_, err := dec.Token()
			return o, err
		case '[':
			list := []any{}
			for dec.More() {
				v, err := parseValue(dec)
				if err != nil {
					return nil, err
				}
				list = append(list, v)
			}
			_, err := dec.Token()
			return list, err
		}
		return nil, fmt.Errorf("unexpected %v", t)
	default:
		return t, nil
	}
}

// encode writes v with two-space indentation, like json.MarshalIndent but
// without escaping <, >, and &.
func encode(v any) string {
	var b strings.Builder
	write(&b, v, "")
	return b.String()
}

func write(b *strings.Builder, v any, indent string) {
	inner := indent + "  "
	switch t := v.(type) {
	case *object:
		if len(t.keys) == 0 {
			b.WriteString("{}")
			return
		}
		b.WriteString("{\n")
		for i, k := range t.keys {
			b.WriteString(inner)
			b.WriteString(quote(k))
			b.WriteString(": ")
			write(b, t.vals[k], inner)
			if i < len(t.keys)-1 {
				b.WriteByte(',')
			}
			b.WriteByte('\n')
		}
		b.WriteString(indent + "}")
	case []any:
		if len(t) == 0 {
			b.WriteString("[]")
			return
		}
		b.WriteString("[\n")
		for i, item := range t {
			b.WriteString(inner)
			write(b, item, inner)
			if i < len(t)-1 {
				b.WriteByte(',')
			}
			b.WriteByte('\n')
		}
		b.WriteString(indent + "]")
	case string:
		b.WriteString(quote(t))
	case json.Number:
		b.WriteString(t.String())
	case int:
		fmt.Fprint(b, t)
	case bool:
		fmt.Fprint(b, t)
	case nil:
		b.WriteString("null")
	}
}

func quote(s string) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	return strings.TrimSuffix(buf.String(), "\n")
}
