//go:build !codegen

package usage

import (
	"github.com/fil-forge/ucantone/binding"
	"github.com/fil-forge/ucantone/ucan/command"
)

// Report is the `/s3/usage/report` command. An S3 gateway invokes it on its
// own authority (issuer = subject = the gateway) with the service that keeps
// usage as the audience. The counts are the gateway's own account of what it
// did, so the invocation carries no proofs: the service decides whether to
// accept a report by who issued it.
//
// A report is safe to repeat, and reports may arrive in any order. Each count
// is a total that only grows, and the service keeps the largest it is sent
// for a space and hour. One count the service cannot use does
// not fail the report: it comes back as a [Warning] beside the result.
var Report = binding.Bind[*ReportArguments, *ReportOK](command.MustParse("/s3/usage/report"))
