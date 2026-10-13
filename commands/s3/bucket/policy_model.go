//go:build !codegen

// The codec below calls StatementModel's generated methods, which the
// generator builds the package without.

package bucket

import (
	"io"
	"slices"

	"github.com/fil-forge/libforge/commands"
)

// Model returns the IPLD form of st: its lists as sets, a wildcard principal
// as the single member [Wildcard], stray ids beside it dropped.
func (st Statement) Model() StatementModel {
	m := StatementModel{
		Sid:       st.Sid,
		Effect:    st.Effect,
		Principal: set(st.Principal.IDs),
		Action:    set(st.Actions),
		Resource:  set(st.Resources),
	}
	if st.Principal.All {
		m.Principal = map[string]commands.Unit{Wildcard: {}}
	}
	return m
}

// Statement returns the statement m encodes: its members sorted, lists never
// nil, and [Wildcard] among the principals meaning [Everyone].
func (m StatementModel) Statement() Statement {
	st := Statement{Sid: m.Sid, Effect: m.Effect, Actions: members(m.Action), Resources: members(m.Resource)}
	if _, all := m.Principal[Wildcard]; all {
		st.Principal = Everyone()
	} else {
		st.Principal = Only(members(m.Principal)...)
	}
	return st
}

func set(list []string) map[string]commands.Unit {
	s := make(map[string]commands.Unit, len(list))
	for _, m := range list {
		s[m] = commands.Unit{}
	}
	return s
}

func members(s map[string]commands.Unit) []string {
	out := make([]string, 0, len(s))
	for m := range s {
		out = append(out, m)
	}
	slices.Sort(out)
	return out
}

func (st *Statement) model() *StatementModel {
	if st == nil {
		return nil
	}
	m := st.Model()
	return &m
}

// MarshalCBOR writes st as its [StatementModel].
func (st *Statement) MarshalCBOR(w io.Writer) error { return st.model().MarshalCBOR(w) }

// UnmarshalCBOR reads a [StatementModel] into st.
func (st *Statement) UnmarshalCBOR(r io.Reader) error {
	var m StatementModel
	if err := m.UnmarshalCBOR(r); err != nil {
		return err
	}
	*st = m.Statement()
	return nil
}

// MarshalDagJSON writes st as its [StatementModel].
func (st *Statement) MarshalDagJSON(w io.Writer) error { return st.model().MarshalDagJSON(w) }

// UnmarshalDagJSON reads a [StatementModel] into st.
func (st *Statement) UnmarshalDagJSON(r io.Reader) error {
	var m StatementModel
	if err := m.UnmarshalDagJSON(r); err != nil {
		return err
	}
	*st = m.Statement()
	return nil
}
