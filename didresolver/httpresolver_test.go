package didresolver_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/fil-forge/libforge/didresolver"
	"github.com/fil-forge/ucantone/did"
	"github.com/fil-forge/ucantone/principal"
	"github.com/stretchr/testify/require"
)

func TestNewHTTPResolver(t *testing.T) {
	t.Run("creates resolver with default timeout", func(t *testing.T) {
		resolver, err := didresolver.NewHTTPResolver()
		require.NoError(t, err)
		require.NotNil(t, resolver)
	})

	t.Run("creates resolver with custom timeout", func(t *testing.T) {
		resolver, err := didresolver.NewHTTPResolver(didresolver.WithTimeout(5*time.Second), didresolver.InsecureResolution())
		require.NoError(t, err)
		require.NotNil(t, resolver)
	})

	t.Run("fails with zero timeout", func(t *testing.T) {
		resolver, err := didresolver.NewHTTPResolver(didresolver.WithTimeout(0))
		require.Error(t, err)
		require.Contains(t, err.Error(), "timeout cannot be zero")
		require.Nil(t, resolver)
	})
}

func TestHTTPResolver_ResolveDIDKey(t *testing.T) {
	testCases := []struct {
		name           string
		setupServer    func() *httptest.Server
		setupGlobbing  func(serverURL string) []string
		inputDID       string
		expectedDIDKey string
		expectError    bool
		errorContains  string
	}{
		{
			name: "successful resolution",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != didresolver.WellKnownDIDPath {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					doc := didresolver.Document{
						Context: []string{"https://w3id.org/did/v1"},
						ID:      "did:web:example.com",
						VerificationMethod: []didresolver.VerificationMethod{
							{
								ID:                 "did:web:example.com#key1",
								Type:               "Ed25519VerificationKey2018",
								Controller:         "did:web:example.com",
								PublicKeyMultibase: "z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK",
							},
						},
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(doc)
				}))
			},
			inputDID:       "", // Will be set based on server URL
			expectedDIDKey: "did:key:z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK",
			expectError:    false,
		},
		{
			name: "successful resolution with pattern",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != didresolver.WellKnownDIDPath {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					doc := didresolver.Document{
						Context: []string{"https://w3id.org/did/v1"},
						ID:      "did:web:example.com",
						VerificationMethod: []didresolver.VerificationMethod{
							{
								ID:                 "did:web:example.com#key1",
								Type:               "Ed25519VerificationKey2018",
								Controller:         "did:web:example.com",
								PublicKeyMultibase: "z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK",
							},
						},
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(doc)
				}))
			},
			setupGlobbing: func(serverURL string) []string {
				return []string{"*"}
			},
			inputDID:       "", // Will be set based on server URL
			expectedDIDKey: "did:key:z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK",
			expectError:    false,
		},
		{
			name:        "DID resolution not permitted by pattern",
			setupServer: func() *httptest.Server { return nil },
			setupGlobbing: func(serverURL string) []string {
				return []string{"*.storacha.network"}
			},
			inputDID:      "did:web:notfound.com",
			expectError:   true,
			errorContains: "resolution via HTTP not permitted",
		},
		{
			name:        "invalid domain when matching against glob",
			setupServer: func() *httptest.Server { return nil },
			setupGlobbing: func(serverURL string) []string {
				return []string{"*.storacha.network"}
			},
			// make too long
			inputDID:      fmt.Sprintf("did:web:%s.storacha.network", strings.Repeat("a", 254)),
			expectError:   true,
			errorContains: "invalid DID",
		},
		{
			name: "HTTP error response",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			inputDID:      "", // Will be set based on server URL
			expectError:   true,
			errorContains: "unexpected status: 404",
		},
		{
			name: "invalid JSON response",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != didresolver.WellKnownDIDPath {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte("invalid json"))
				}))
			},
			inputDID:      "", // Will be set based on server URL
			expectError:   true,
			errorContains: "parsing DID document JSON",
		},
		{
			name: "no verification methods",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != didresolver.WellKnownDIDPath {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					doc := didresolver.Document{
						Context:            []string{"https://w3id.org/did/v1"},
						ID:                 "did:web:example.com",
						VerificationMethod: []didresolver.VerificationMethod{},
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(doc)
				}))
			},
			inputDID:      "", // Will be set based on server URL
			expectError:   true,
			errorContains: "missing verificationMethod",
		},
		{
			name: "empty public key",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != didresolver.WellKnownDIDPath {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					doc := didresolver.Document{
						Context: []string{"https://w3id.org/did/v1"},
						ID:      "did:web:example.com",
						VerificationMethod: []didresolver.VerificationMethod{
							{
								ID:                 "did:web:example.com#key1",
								Type:               "Ed25519VerificationKey2018",
								Controller:         "did:web:example.com",
								PublicKeyMultibase: "",
							},
						},
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(doc)
				}))
			},
			inputDID:      "", // Will be set based on server URL
			expectError:   true,
			errorContains: "missing publicKeyMultibase",
		},
		{
			name: "invalid public key format",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != didresolver.WellKnownDIDPath {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					doc := didresolver.Document{
						Context: []string{"https://w3id.org/did/v1"},
						ID:      "did:web:example.com",
						VerificationMethod: []didresolver.VerificationMethod{
							{
								ID:                 "did:web:example.com#key1",
								Type:               "Ed25519VerificationKey2018",
								Controller:         "did:web:example.com",
								PublicKeyMultibase: "invalid-key",
							},
						},
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(doc)
				}))
			},
			inputDID:      "", // Will be set based on server URL
			expectError:   true,
			errorContains: "parsing multibase key",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var server *httptest.Server
			if tc.setupServer != nil {
				server = tc.setupServer()
				if server != nil {
					defer server.Close()
				}
			}

			var serverURL string
			if server != nil {
				serverURL = server.URL
			}
			var patterns []string
			if tc.setupGlobbing != nil {
				patterns = tc.setupGlobbing(serverURL)
			}

			resolver, err := didresolver.NewHTTPResolver(
				didresolver.InsecureResolution(),
				didresolver.WithPatterns(patterns...),
			)
			require.NoError(t, err)

			// For tests where inputDID is empty, derive it from server URL
			var inputDID did.DID
			if tc.inputDID == "" && server != nil {
				u, _ := url.Parse(serverURL)
				inputDID, err = did.Parse("did:web:" + u.Host)
				require.NoError(t, err)
			} else {
				inputDID, err = did.Parse(tc.inputDID)
				require.NoError(t, err)
			}

			result, unresolvedErr := resolver.Resolve(t.Context(), inputDID)

			if tc.expectError {
				require.NotNil(t, unresolvedErr)
				require.Contains(t, unresolvedErr.Error(), "unable to resolve")
				require.Nil(t, result)
				if tc.errorContains != "" {
					require.Contains(t, unresolvedErr.Error(), tc.errorContains)
				}
			} else {
				require.Nil(t, unresolvedErr)
				// The resolver wraps the underlying did:key verifier so it
				// announces the originally-requested DID — required for
				// ucantone token.VerifySignature, which compares the token's
				// issuer DID against the verifier's DID before checking
				// signature bytes.
				require.Equal(t, inputDID, result.DID())
				// expectedDIDKey identifies the underlying did:key the
				// resolver should have extracted from the document; reach
				// through Unwrap() to assert it.
				expectedDIDKey, err := did.Parse(tc.expectedDIDKey)
				require.NoError(t, err)
				unwrapper, ok := result.(interface {
					Unwrap() principal.Verifier
				})
				require.True(t, ok, "resolver should return a wrapped verifier")
				require.Equal(t, expectedDIDKey, unwrapper.Unwrap().DID())
			}
		})
	}
}

func TestHTTPResolver_ResolveDIDKey_Timeout(t *testing.T) {
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer slowServer.Close()

	u, err := url.Parse(slowServer.URL)
	require.NoError(t, err)

	didWeb, err := did.Parse("did:web:" + u.Host)
	require.NoError(t, err)

	resolver, err := didresolver.NewHTTPResolver(didresolver.WithTimeout(50*time.Millisecond), didresolver.InsecureResolution())
	require.NoError(t, err)

	result, unresolvedErr := resolver.Resolve(t.Context(), didWeb)
	require.NotNil(t, unresolvedErr)
	require.Contains(t, unresolvedErr.Error(), "unable to resolve")
	require.Nil(t, result)
}

func TestHTTPResolver_ResolveDIDKey_Context(t *testing.T) {
	requestReceived := make(chan bool, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case requestReceived <- true:
		default:
		}

		if r.URL.Path != didresolver.WellKnownDIDPath {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		doc := didresolver.Document{
			Context: []string{"https://w3id.org/did/v1"},
			ID:      "did:web:example.com",
			VerificationMethod: []didresolver.VerificationMethod{
				{
					ID:                 "did:web:example.com#key1",
					Type:               "Ed25519VerificationKey2018",
					Controller:         "did:web:example.com",
					PublicKeyMultibase: "z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(doc)
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	require.NoError(t, err)

	didWeb, err := did.Parse("did:web:" + u.Host)
	require.NoError(t, err)

	resolver, err := didresolver.NewHTTPResolver(didresolver.InsecureResolution())
	require.NoError(t, err)

	result, unresolvedErr := resolver.Resolve(t.Context(), didWeb)
	require.Nil(t, unresolvedErr)
	require.NotEqual(t, did.Undef, result)

	select {
	case <-requestReceived:
	case <-time.After(time.Second):
		t.Fatal("request was not received by server")
	}
}

func TestFlexibleContext_UnmarshalJSON(t *testing.T) {
	testCases := []struct {
		name          string
		input         string
		expectedValue didresolver.FlexibleContext
		expectError   bool
		errorContains string
	}{
		{
			name:          "single string context",
			input:         `"https://w3id.org/did/v1"`,
			expectedValue: didresolver.FlexibleContext{"https://w3id.org/did/v1"},
			expectError:   false,
		},
		{
			name:          "array of strings context",
			input:         `["https://w3id.org/did/v1", "https://w3id.org/security/v1"]`,
			expectedValue: didresolver.FlexibleContext{"https://w3id.org/did/v1", "https://w3id.org/security/v1"},
			expectError:   false,
		},
		{
			name:          "empty array context",
			input:         `[]`,
			expectedValue: didresolver.FlexibleContext{},
			expectError:   false,
		},
		{
			name:          "invalid type - number",
			input:         `123`,
			expectError:   true,
			errorContains: "@context must be string or array of strings",
		},
		{
			name:          "invalid type - object",
			input:         `{"foo": "bar"}`,
			expectError:   true,
			errorContains: "@context must be string or array of strings",
		},
		{
			name:          "invalid type - boolean",
			input:         `true`,
			expectError:   true,
			errorContains: "@context must be string or array of strings",
		},
		{
			name:          "array with non-string elements",
			input:         `["https://w3id.org/did/v1", 123]`,
			expectError:   true,
			errorContains: "@context must be string or array of strings",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var fc didresolver.FlexibleContext
			err := json.Unmarshal([]byte(tc.input), &fc)

			if tc.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errorContains)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedValue, fc)
			}
		})
	}
}

func TestHTTPResolver_ResolveDIDKey_ContextFormats(t *testing.T) {
	testCases := []struct {
		name           string
		setupServer    func() *httptest.Server
		expectedDIDKey string
	}{
		{
			name: "DID document with string context",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != didresolver.WellKnownDIDPath {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					// Using raw JSON to ensure we send a string context, not array
					docJSON := `{
						"@context": "https://w3id.org/did/v1",
						"id": "did:web:example.com",
						"verificationMethod": [{
							"id": "did:web:example.com#key1",
							"type": "Ed25519VerificationKey2018",
							"controller": "did:web:example.com",
							"publicKeyMultibase": "z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK"
						}]
					}`
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(docJSON))
				}))
			},
			expectedDIDKey: "did:key:z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK",
		},
		{
			name: "DID document with array context",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != didresolver.WellKnownDIDPath {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					doc := didresolver.Document{
						Context: didresolver.FlexibleContext{"https://w3id.org/did/v1", "https://w3id.org/security/v1"},
						ID:      "did:web:example.com",
						VerificationMethod: []didresolver.VerificationMethod{
							{
								ID:                 "did:web:example.com#key1",
								Type:               "Ed25519VerificationKey2018",
								Controller:         "did:web:example.com",
								PublicKeyMultibase: "z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK",
							},
						},
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(doc)
				}))
			},
			expectedDIDKey: "did:key:z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			server := tc.setupServer()
			defer server.Close()

			u, err := url.Parse(server.URL)
			require.NoError(t, err)

			didWeb, err := did.Parse("did:web:" + u.Host)
			require.NoError(t, err)

			resolver, err := didresolver.NewHTTPResolver(didresolver.InsecureResolution())
			require.NoError(t, err)

			result, unresolvedErr := resolver.Resolve(t.Context(), didWeb)
			require.Nil(t, unresolvedErr)

			// Resolver wraps the underlying did:key as the requested did:web —
			// see ucantone/ucan/token/token.go for why this matters.
			require.Equal(t, didWeb, result.DID())
			expectedDIDKey, err := did.Parse(tc.expectedDIDKey)
			require.NoError(t, err)
			unwrapper, ok := result.(interface {
				Unwrap() principal.Verifier
			})
			require.True(t, ok, "resolver should return a wrapped verifier")
			require.Equal(t, expectedDIDKey, unwrapper.Unwrap().DID())
		})
	}
}

func TestExtractDomainFromDID(t *testing.T) {
	testCases := []struct {
		name           string
		did            string
		expectedDomain string
		expectError    bool
		errorContains  string
	}{
		{
			name:           "valid did:web",
			did:            "did:web:example.com",
			expectedDomain: "example.com",
			expectError:    false,
		},
		{
			name:           "valid did:web with subdomain",
			did:            "did:web:api.example.com",
			expectedDomain: "api.example.com",
			expectError:    false,
		},
		{
			name:          "invalid prefix",
			did:           "did:key:z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK",
			expectError:   true,
			errorContains: "invalid DID web format: must start with 'did:web:'",
		},
		{
			name:          "empty domain",
			did:           "did:web:",
			expectError:   true,
			errorContains: "invalid DID web format: no domain specified",
		},
		{
			name:          "domain too long",
			did:           "did:web:" + strings.Repeat("a", 254),
			expectError:   true,
			errorContains: "domain too long",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			did, err := did.Parse(tc.did)
			require.NoError(t, err)

			domain, err := didresolver.ExtractDomainFromDID(did)

			if tc.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errorContains)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedDomain, domain)
			}
		})
	}
}
