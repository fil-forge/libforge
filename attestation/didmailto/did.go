package didmailto

import (
	"fmt"
	"net/mail"
	"net/url"
	"strings"

	"github.com/fil-forge/ucantone/did"
	"github.com/fil-forge/ucantone/errors"
)

const InvalidMailtoDIDErrorName = "InvalidMailtoDID"

var ErrInvalidMailtoDID = errors.New(InvalidMailtoDIDErrorName, "invalid mailto DID")

// New creates a new mailto DID from an RFC 5322 formatted email address:
// user@example.com is did:mailto:example.com:user.
//
// A DID identifier holds only letters, digits, ".", "-", "_" and
// percent-escapes, so every other character of the address is percent-encoded:
// user+tag@example.com is did:mailto:example.com:user%2Btag.
func New(email string) (did.DID, error) {
	a, err := mail.ParseAddress(email)
	if err != nil {
		return did.DID{}, fmt.Errorf("parsing email: %w", err)
	}
	at := strings.LastIndex(a.Address, "@")
	var local, domain string
	if at < 0 {
		// This is a malformed address ("@" is required in addr-spec);
		return did.DID{}, fmt.Errorf("malformed email address: %s", email)
	}
	local, domain = a.Address[:at], a.Address[at+1:]
	return did.Parse(fmt.Sprintf("did:%s:%s:%s", Method, escape(domain), escape(local)))
}

// escape percent-encodes every character of s that a DID identifier cannot
// carry unescaped.
// https://www.w3.org/TR/did-core/#did-syntax
func escape(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' || c == '.' || c == '-' || c == '_' {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

// Email extracts the email address from the DID.
func Email(d did.DID) (string, error) {
	if !strings.HasPrefix(d.String(), "did:mailto:") {
		return "", ErrInvalidMailtoDID
	}
	parts := strings.Split(d.String(), ":")
	if len(parts) != 4 {
		return "", ErrInvalidMailtoDID
	}
	emailLocal, err := url.QueryUnescape(parts[3])
	if err != nil {
		return "", ErrInvalidMailtoDID
	}
	domain, err := url.QueryUnescape(parts[2])
	if err != nil {
		return "", ErrInvalidMailtoDID
	}
	return fmt.Sprintf("%s@%s", emailLocal, domain), nil
}

func Parse(str string) (did.DID, error) {
	d, err := did.Parse(str)
	if err != nil {
		return did.DID{}, err
	}
	if !strings.HasPrefix(d.String(), "did:mailto:") {
		return did.DID{}, ErrInvalidMailtoDID
	}
	return d, nil
}

func Format(d did.DID) string {
	return d.String()
}
