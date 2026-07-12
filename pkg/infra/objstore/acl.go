package objstore

import (
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// ObjectACLHeader is the S3-compatible ACL request header.
// KS3 also accepts the vendor-specific "x-kss-acl" header with the same value.
const ObjectACLHeader = "x-amz-acl"

// ObjectCannedACL enumerates supported canned ACL values.
type ObjectCannedACL = types.ObjectCannedACL

const (
	ObjectACLPrivate               = types.ObjectCannedACLPrivate
	ObjectACLPublicRead            = types.ObjectCannedACLPublicRead
	ObjectACLPublicReadWrite       = types.ObjectCannedACLPublicReadWrite
	ObjectACLAuthenticatedRead     = types.ObjectCannedACLAuthenticatedRead
	ObjectACLBucketOwnerRead       = types.ObjectCannedACLBucketOwnerRead
	ObjectACLBucketOwnerFullControl = types.ObjectCannedACLBucketOwnerFullControl
)

// NormalizeObjectACL normalizes common ACL aliases to canned ACL values.
func NormalizeObjectACL(acl string) string {
	switch strings.ToLower(strings.TrimSpace(acl)) {
	case "", "private":
		return string(ObjectACLPrivate)
	case "public-read":
		return string(ObjectACLPublicRead)
	case "public-read-write":
		return string(ObjectACLPublicReadWrite)
	case "authenticated-read":
		return string(ObjectACLAuthenticatedRead)
	case "bucket-owner-read":
		return string(ObjectACLBucketOwnerRead)
	case "bucket-owner-full-control":
		return string(ObjectACLBucketOwnerFullControl)
	default:
		return strings.TrimSpace(acl)
	}
}

// IsPublicObjectACL reports whether objects uploaded with the ACL are anonymously readable.
func IsPublicObjectACL(acl string) bool {
	switch NormalizeObjectACL(acl) {
	case string(ObjectACLPublicRead), string(ObjectACLPublicReadWrite):
		return true
	default:
		return false
	}
}

func resolvePresignPutACL(opts PresignPutOptions) (types.ObjectCannedACL, bool) {
	if strings.TrimSpace(opts.ACL) == "" {
		return "", false
	}
	acl := types.ObjectCannedACL(NormalizeObjectACL(opts.ACL))
	if acl == "" {
		return "", false
	}
	// Private is the default object ACL; omit from presign unless explicitly overridden elsewhere.
	if acl == ObjectACLPrivate {
		return "", false
	}
	return acl, true
}

func presignPutRequiredHeaders(opts PresignPutOptions) map[string]string {
	headers := make(map[string]string)
	if acl, ok := resolvePresignPutACL(opts); ok {
		headers[ObjectACLHeader] = string(acl)
	}
	if contentType := strings.TrimSpace(opts.ContentType); contentType != "" {
		headers["Content-Type"] = contentType
	}
	if len(headers) == 0 {
		return nil
	}
	return headers
}
