package yahoo

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

// DecodeWarning records a non-fatal failure to parse a numeric field from a
// Yahoo response. Yahoo returns all values as strings; when a non-empty value
// cannot be parsed, the converted field falls back to its zero value and a
// DecodeWarning is attached to the enclosing model. This lets callers tell a
// genuine zero apart from malformed or unexpected data instead of silently
// treating both as zero.
type DecodeWarning struct {
	Field string // logical field name, e.g. "player_points.total"
	Value string // the raw string that failed to parse
	Err   error  // the underlying strconv error
}

func (w DecodeWarning) String() string {
	return fmt.Sprintf("%s: cannot parse %q: %v", w.Field, w.Value, w.Err)
}

// decodeWarningJSON is the stable wire form. The Err interface has no reliable
// JSON representation, so it is flattened to its message on the way out and
// rebuilt as a plain error on the way in — this keeps a model carrying
// DecodeWarnings round-trippable through the cache (a malformed value is exactly
// when a warning exists, so an unmarshalable warning would silently defeat
// caching precisely when it matters).
type decodeWarningJSON struct {
	Field string `json:"field"`
	Value string `json:"value"`
	Error string `json:"error,omitempty"`
}

func (w DecodeWarning) MarshalJSON() ([]byte, error) {
	out := decodeWarningJSON{Field: w.Field, Value: w.Value}
	if w.Err != nil {
		out.Error = w.Err.Error()
	}
	return json.Marshal(out)
}

func (w *DecodeWarning) UnmarshalJSON(data []byte) error {
	var in decodeWarningJSON
	if err := json.Unmarshal(data, &in); err != nil {
		return err
	}
	w.Field, w.Value = in.Field, in.Value
	if in.Error != "" {
		w.Err = errors.New(in.Error)
	} else {
		w.Err = nil
	}
	return nil
}

// decoder parses Yahoo's string-encoded numbers and accumulates a warning for
// each non-empty value that fails to parse. An empty string is treated as an
// absent field (zero value, no warning), matching prior behaviour.
type decoder struct {
	warnings []DecodeWarning
}

func (d *decoder) atoi(field, s string) int {
	if s == "" {
		return 0
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		d.warnings = append(d.warnings, DecodeWarning{Field: field, Value: s, Err: err})
		return 0
	}
	return v
}

func (d *decoder) parseFloat(field, s string) float64 {
	if s == "" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		d.warnings = append(d.warnings, DecodeWarning{Field: field, Value: s, Err: err})
		return 0
	}
	return v
}

func (d *decoder) parseInt64(field, s string) int64 {
	if s == "" {
		return 0
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		d.warnings = append(d.warnings, DecodeWarning{Field: field, Value: s, Err: err})
		return 0
	}
	return v
}

// merge appends another set of warnings under a prefix so nested warnings keep
// context, e.g. "teams[1].team_points.total".
func (d *decoder) merge(prefix string, ws []DecodeWarning) {
	for _, w := range ws {
		w.Field = prefix + "." + w.Field
		d.warnings = append(d.warnings, w)
	}
}
