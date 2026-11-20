package egress

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/ipfs/go-cid"
)

// BatchPlaceholder is the text in [TrackArguments.Endpoint] that stands for
// the batch CID.
const BatchPlaceholder = "{cid}"

// TrackArguments is the argument shape of `/egress/track`.
type TrackArguments struct {
	// Receipts is the CID of the batch of receipts the reporter has staged.
	Receipts cid.Cid `cborgen:"receipts" dagjsongen:"receipts"`
	// Endpoint is the URL template the batch can be fetched from, with
	// [BatchPlaceholder] where the batch CID goes. It is a template and not a
	// URL until the placeholder is replaced: see [TrackArguments.BatchURL].
	Endpoint string `cborgen:"endpoint" dagjsongen:"endpoint"`
}

// BatchURL returns the URL the batch can be fetched from: Endpoint with its
// placeholder replaced by the batch CID.
func (a TrackArguments) BatchURL() (*url.URL, error) {
	if !a.Receipts.Defined() {
		return nil, errors.New("track arguments name no batch")
	}
	return ResolveEndpoint(a.Endpoint, a.Receipts)
}

// ResolveEndpoint replaces the placeholder in an endpoint template with the
// batch CID and parses the result. The template must hold the placeholder and
// resolve to an absolute http or https URL.
func ResolveEndpoint(endpoint string, batch cid.Cid) (*url.URL, error) {
	if !strings.Contains(endpoint, BatchPlaceholder) {
		return nil, fmt.Errorf("endpoint %q has no %s placeholder", endpoint, BatchPlaceholder)
	}
	u, err := url.Parse(strings.ReplaceAll(endpoint, BatchPlaceholder, batch.String()))
	if err != nil {
		return nil, fmt.Errorf("parsing batch URL: %w", err)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("endpoint %q is not an absolute http or https URL", endpoint)
	}
	return u, nil
}

// TrackOK is the success return for `/egress/track`. The response also carries
// the `/egress/consolidate` invocation that will process the batch.
type TrackOK struct{}

// ConsolidateArguments is the argument shape of `/egress/consolidate`.
type ConsolidateArguments struct {
	// Cause is the task link of the `/egress/track` invocation that submitted
	// the batch.
	Cause cid.Cid `cborgen:"cause" dagjsongen:"cause"`
}

// ConsolidateOK is the success return for `/egress/consolidate`.
type ConsolidateOK struct {
	// TotalEgress is the bytes of egress accounted for by the receipts that
	// were processed.
	TotalEgress uint64 `cborgen:"totalEgress" dagjsongen:"totalEgress"`
	// Errors lists the receipts that could not be processed. It is empty when
	// every receipt in the batch was.
	Errors []ReceiptError `cborgen:"errors" dagjsongen:"errors"`
}

// ReceiptError describes a receipt in a batch that could not be processed.
type ReceiptError struct {
	// Name is the machine readable name of the error.
	Name string `cborgen:"name" dagjsongen:"name"`
	// Message is a human readable description of the error.
	Message string `cborgen:"message" dagjsongen:"message"`
	// Receipt is the link of the receipt that failed.
	Receipt cid.Cid `cborgen:"receipt" dagjsongen:"receipt"`
}
