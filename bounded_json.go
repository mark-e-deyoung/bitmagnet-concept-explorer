package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

// decodeBoundedJSON reads at most maxBytes+1 before parsing. A valid prefix
// followed by excessive whitespace or truncated trailing data is rejected,
// rather than being misclassified as an ordinary EOF by a limited decoder.
// This only handles upstream responses; request-body limits are independent.
func decodeBoundedJSON(body io.Reader, maxBytes int, dst any) error {
	if maxBytes <= 0 {
		return errors.New("invalid JSON budget")
	}
	buf, e := io.ReadAll(io.LimitReader(body, int64(maxBytes)+1))
	if e != nil {
		return errors.New("upstream JSON body read failed")
	}
	if len(buf) > maxBytes {
		return errors.New("upstream JSON response exceeds byte budget")
	}
	d := json.NewDecoder(bytes.NewReader(buf))
	if e := d.Decode(dst); e != nil {
		return errors.New("upstream JSON response is invalid")
	}
	if e := d.Decode(new(any)); e != io.EOF {
		return errors.New("upstream JSON response has trailing content")
	}
	return nil
}
