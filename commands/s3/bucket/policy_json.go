package bucket

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Wildcard is the principal "*", naming every principal of the tenant.
const Wildcard = "*"

// UserARNPrefix is the prefix of a principal's user ARN; the principal id
// follows it.
const UserARNPrefix = "arn:aws:iam:::user/"

// UserARN returns the user ARN of principal id.
func UserARN(id string) string { return UserARNPrefix + id }

// Everyone returns the principal set naming every principal of the tenant.
func Everyone() Principal { return Principal{All: true} }

// Only returns the principal set naming exactly the given ids.
func Only(ids ...string) Principal { return Principal{IDs: ids} }

// MarshalJSON encodes All as "*" and anything else as {"AWS": [user ARNs]},
// nil encoding as an empty list.
func (p Principal) MarshalJSON() ([]byte, error) {
	if p.All {
		return json.Marshal(Wildcard)
	}
	arns := make([]string, len(p.IDs))
	for i, id := range p.IDs {
		arns[i] = UserARN(id)
	}
	return json.Marshal(struct {
		AWS []string `json:"AWS"`
	}{arns})
}

// UnmarshalJSON accepts "*", {"AWS": "*"}, {"AWS": "<user ARN>"} or
// {"AWS": ["<user ARN>", ...]}, keeping each id once, in the order it first
// appears.
func (p *Principal) UnmarshalJSON(data []byte) error {
	const shape = `must be "*" or {"AWS": user ARNs}`
	data = bytes.TrimSpace(data)
	if len(data) > 0 && data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return fmt.Errorf("Principal: %w", err)
		}
		if s != Wildcard {
			return fmt.Errorf("Principal %q: a string principal must be %q", s, Wildcard)
		}
		*p = Everyone()
		return nil
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil || obj == nil {
		return fmt.Errorf("Principal: %s", shape)
	}
	for k := range obj {
		if k != "AWS" {
			return fmt.Errorf("Principal: unknown key %q, %s", k, shape)
		}
	}
	aws, ok := obj["AWS"]
	if !ok {
		return fmt.Errorf("Principal: %s", shape)
	}
	var one string
	if json.Unmarshal(aws, &one) == nil && one == Wildcard {
		*p = Everyone()
		return nil
	}
	var arns stringOrList
	if err := json.Unmarshal(aws, &arns); err != nil {
		return fmt.Errorf("Principal: %s", shape)
	}
	ids := make([]string, len(arns))
	for i, arn := range arns {
		if arn == Wildcard {
			return fmt.Errorf("Principal %q must be the bare string or {\"AWS\": %q}, not a list entry", Wildcard, Wildcard)
		}
		id, ok := strings.CutPrefix(arn, UserARNPrefix)
		if !ok || id == "" {
			return fmt.Errorf("Principal %q: must be a user ARN %q", arn, UserARN("<id>"))
		}
		ids[i] = id
	}
	*p = Principal{IDs: dedupe(ids)}
	return nil
}

// stringOrList is a JSON field AWS allows as a string or a list of strings.
type stringOrList []string

func (l *stringOrList) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) > 0 && data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		*l = stringOrList{s}
		return nil
	}
	var ss []string
	if err := json.Unmarshal(data, &ss); err != nil {
		return errors.New("must be a string or a list of strings")
	}
	*l = ss
	return nil
}

// dedupe keeps the first occurrence of each id.
func dedupe(ids []string) []string {
	seen := make(map[string]bool, len(ids))
	out := ids[:0]
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}
