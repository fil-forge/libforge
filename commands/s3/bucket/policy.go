//go:build !codegen

package bucket

import (
	"github.com/fil-forge/ucantone/binding"
	"github.com/fil-forge/ucantone/ucan/command"
)

// Policy is the `/s3/bucket/policy` command. Ingot invokes it on Hilt with a
// forwarded S3 GetBucketPolicy, PutBucketPolicy or DeleteBucketPolicy request;
// the request's method selects the operation.
var Policy = binding.Bind[*PolicyArguments, *PolicyOK](command.MustParse("/s3/bucket/policy"))
