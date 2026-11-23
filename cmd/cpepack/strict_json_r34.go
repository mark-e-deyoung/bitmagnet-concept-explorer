// Strict byte-first admission for the research-only offline CPE CLI.
// Token recursion checks duplicate object member names at all nesting depths.
// This does not define canonical CPE concept/sweep identity or accepted policy.
package main

import (
    "bytes"
    "encoding/json"
    "errors"
    "io"
    "unicode/utf8"
)

const r34MaxJSONDepth = 128

func strictJSONAdmission(raw []byte) error {
    if !utf8.Valid(raw) {
        return errors.New("invalid_json_utf8")
    }
    d := json.NewDecoder(bytes.NewReader(raw))
    if err := strictWalkJSON(d, 0); err != nil {
        return err
    }
    if _, err := d.Token(); err != io.EOF {
        if err != nil {
            return errors.New("malformed_json_trailer")
        }
        return errors.New("trailing_json_value")
    }
    return nil
}

func strictWalkJSON(d *json.Decoder, depth int) error {
    if depth > r34MaxJSONDepth {
        return errors.New("json_depth_limit")
    }
    tok, err := d.Token()
    if err != nil {
        return errors.New("malformed_json_token")
    }
    delim, ok := tok.(json.Delim)
    if !ok {
        return nil
    }
    switch delim {
    case '{':
        seen := make(map[string]struct{})
        for d.More() {
            keyToken, err := d.Token()
            if err != nil {
                return errors.New("malformed_json_member_key")
            }
            key, ok := keyToken.(string)
            if !ok {
                return errors.New("malformed_json_member_key")
            }
            if _, duplicate := seen[key]; duplicate {
                return errors.New("duplicate_json_key")
            }
            seen[key] = struct{}{}
            if err := strictWalkJSON(d, depth+1); err != nil {
                return err
            }
        }
        closeToken, err := d.Token()
        if err != nil || closeToken != json.Delim('}') {
            return errors.New("malformed_json_object")
        }
    case '[':
        for d.More() {
            if err := strictWalkJSON(d, depth+1); err != nil {
                return err
            }
        }
        closeToken, err := d.Token()
        if err != nil || closeToken != json.Delim(']') {
            return errors.New("malformed_json_array")
        }
    default:
        return errors.New("unexpected_json_delimiter")
    }
    return nil
}
