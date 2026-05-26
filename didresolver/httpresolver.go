package didresolver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/fil-forge/ucantone/did"
	"github.com/fil-forge/ucantone/principal/ed25519/verifier"
	pverifier "github.com/fil-forge/ucantone/principal/verifier"
	"github.com/fil-forge/ucantone/ucan"
	verrs "github.com/fil-forge/ucantone/validator/errors"
	"github.com/gobwas/glob"
)

// FlexibleContext handles both string and []string formats for @context field
// as allowed by the DID Core specification
type FlexibleContext []string

func (fc *FlexibleContext) UnmarshalJSON(data []byte) error {
	// Try array first (most common format)
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*fc = FlexibleContext(arr)
		return nil
	}

	// Fall back to single string format
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*fc = FlexibleContext([]string{str})
		return nil
	}

	return fmt.Errorf("@context must be string or array of strings")
}

// Document is a did document that describes a did subject.
// See https://www.w3.org/TR/did-core/#dfn-did-documents.
// Copied from: https://github.com/storacha/indexing-service/blob/fe8f2211a15d851f2672bfeb64dcfc65c52e6011/pkg/server/server.go#L238
type Document struct {
	Context            FlexibleContext      `json:"@context"` // https://w3id.org/did/v1
	ID                 string               `json:"id"`
	Controller         []string             `json:"controller,omitempty"`
	VerificationMethod []VerificationMethod `json:"verificationMethod,omitempty"`
	Authentication     []string             `json:"authentication,omitempty"`
	AssertionMethod    []string             `json:"assertionMethod,omitempty"`
}

// VerificationMethod describes how to authenticate or authorize interactions
// with a did subject.
// See https://www.w3.org/TR/did-core/#dfn-verification-method.
type VerificationMethod struct {
	ID                 string `json:"id,omitempty"`
	Type               string `json:"type,omitempty"`
	Controller         string `json:"controller,omitempty"`
	PublicKeyMultibase string `json:"publicKeyMultibase,omitempty"`
}

type HTTPResolver struct {
	cfg config
}

type config struct {
	timeout  time.Duration
	insecure bool
	globs    map[string]glob.Glob
}

type Option func(*config) error

func WithTimeout(timeout time.Duration) Option {
	return func(c *config) error {
		if timeout == 0 {
			return fmt.Errorf("timeout cannot be zero")
		}
		c.timeout = timeout
		return nil
	}
}

func InsecureResolution() Option {
	return func(c *config) error {
		c.insecure = true
		return nil
	}
}

// WithPatterns restricts resolving of did:web's that match the provided glob
// pattern(s).
//
// Note: the pattern should not include the "did:web:" prefix.
func WithPatterns(patterns ...string) Option {
	return func(c *config) error {
		for _, p := range patterns {
			g, err := glob.Compile(p)
			if err != nil {
				return fmt.Errorf("compiling pattern %q: %w", p, err)
			}
			if c.globs == nil {
				c.globs = map[string]glob.Glob{}
			}
			c.globs[p] = g
		}
		return nil
	}
}

// ExtractDomainFromDID extracts the domain from a DID web string
func ExtractDomainFromDID(didWeb did.DID) (string, error) {
	// Check if it starts with the required prefix
	if didWeb.Method() != "web" {
		return "", fmt.Errorf("invalid DID web format: must start with 'did:web:'")
	}

	// Extract the domain part
	domain := didWeb.Identifier()

	// Check if domain is empty
	if domain == "" {
		return "", fmt.Errorf("invalid DID web format: no domain specified")
	}

	// Validate the domain format
	if err := validateDomain(domain); err != nil {
		return "", fmt.Errorf("invalid domain '%s': %w", domain, err)
	}

	return domain, nil
}

// validateDomain checks if a string is a valid domain name
func validateDomain(domain string) error {
	// Basic length check
	if len(domain) > 253 {
		return fmt.Errorf("domain too long (max 253 characters)")
	}

	// TODO we could do further checking that the domain is valid, length seems fine for now.

	return nil
}

func WellKnownEndpointFromDID(didWeb did.DID, insecure bool) (url.URL, error) {
	domain, err := ExtractDomainFromDID(didWeb)
	if err != nil {
		return url.URL{}, err
	}

	schema := "https"
	if insecure {
		schema = "http"
	}

	endpoint := url.URL{
		Scheme: schema,
		Host:   domain,
		Path:   WellKnownDIDPath,
	}

	if _, err := url.Parse(endpoint.String()); err != nil {
		return url.URL{}, fmt.Errorf("invalid did domain: %w", err)
	}

	return endpoint, nil
}

const WellKnownDIDPath = "/.well-known/did.json"

func NewHTTPResolver(options ...Option) (*HTTPResolver, error) {
	cfg := &config{
		timeout:  10 * time.Second,
		insecure: false,
	}
	for _, opt := range options {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}
	// default timeout of 10 seconds, options can override
	return &HTTPResolver{cfg: *cfg}, nil
}

func (r *HTTPResolver) Resolve(ctx context.Context, input did.DID) (ucan.Verifier, error) {
	if r.cfg.globs != nil {
		match := false
		for _, g := range r.cfg.globs {
			if match = g.Match(input.Identifier()); match {
				break
			}
		}
		if !match {
			return nil, verrs.NewDIDKeyResolutionError(input, fmt.Errorf("resolution via HTTP not permitted"))
		}
	}

	endpoint, err := WellKnownEndpointFromDID(input, r.cfg.insecure)
	if err != nil {
		return nil, verrs.NewDIDKeyResolutionError(input, fmt.Errorf("invalid DID: %w", err))
	}

	ctx, cancel := context.WithTimeout(ctx, r.cfg.timeout)
	defer cancel()
	didDoc, err := fetchDIDDocument(ctx, endpoint)
	if err != nil {
		return nil, verrs.NewDIDKeyResolutionError(input, fmt.Errorf("failed to fetch DID document: %w", err))
	}
	if len(didDoc.VerificationMethod) == 0 {
		return nil, verrs.NewDIDKeyResolutionError(input, fmt.Errorf("missing verificationMethod in DID document"))
	}

	pubKeyStr := didDoc.VerificationMethod[0].PublicKeyMultibase
	if pubKeyStr == "" {
		return nil, verrs.NewDIDKeyResolutionError(input, fmt.Errorf("missing publicKeyMultibase in DID document"))
	}

	// TODO: multiple verification methods when https://github.com/fil-forge/ucantone/pull/7 lands
	didKey, err := verifier.Parse(fmt.Sprintf("did:key:%s", pubKeyStr))
	if err != nil {
		return nil, verrs.NewDIDKeyResolutionError(input, fmt.Errorf("parsing multibase key: %w", err))
	}

	// token.VerifySignature compares the token's Issuer DID against the
	// verifier's DID — if the issuer is did:web:foo and we return an unwrapped
	// did:key verifier, that equality check fails and the signature is
	// rejected before the bytes are even examined. Wrap so the verifier
	// announces the originally-requested DID.
	wrapped, err := pverifier.Wrap(didKey, input)
	if err != nil {
		return nil, verrs.NewDIDKeyResolutionError(input, fmt.Errorf("wrapping verifier as %s: %w", input, err))
	}
	return wrapped, nil
}

func fetchDIDDocument(ctx context.Context, endpoint url.URL) (*Document, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating HTTP request: %w", err)
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var didDoc Document
	if err := json.Unmarshal(body, &didDoc); err != nil {
		return nil, fmt.Errorf("parsing DID document JSON: %w", err)
	}

	return &didDoc, nil
}
