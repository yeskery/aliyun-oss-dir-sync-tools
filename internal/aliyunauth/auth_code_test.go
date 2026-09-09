package aliyunauth

import (
	"encoding/base64"
	"testing"
)

func TestParseAuthCode(t *testing.T) {
	rawJSON := `{
		"id": "STS.LTAI5tMockID123",
		"secret": "MockSecretKey456",
		"stoken": "MockToken789",
		"eptpl": "https://{bucket}.oss-cn-beijing.aliyuncs.com",
		"osspath": "oss://my-test-bucket/images/"
	}`

	b64 := base64.StdEncoding.EncodeToString([]byte(rawJSON))

	parsed, err := ParseAuthCode(b64)
	if err != nil {
		t.Fatalf("ParseAuthCode failed: %v", err)
	}

	if parsed.AccessKeyID != "STS.LTAI5tMockID123" {
		t.Errorf("unexpected AK: %s", parsed.AccessKeyID)
	}
	if parsed.AccessKeySecret != "MockSecretKey456" {
		t.Errorf("unexpected SK: %s", parsed.AccessKeySecret)
	}
	if parsed.SecurityToken != "MockToken789" {
		t.Errorf("unexpected STS: %s", parsed.SecurityToken)
	}
	if parsed.Endpoint != "oss-cn-beijing.aliyuncs.com" {
		t.Errorf("unexpected Endpoint: %s", parsed.Endpoint)
	}
	if parsed.BucketName != "my-test-bucket" {
		t.Errorf("unexpected Bucket: %s", parsed.BucketName)
	}
	if parsed.PathPrefix != "images/" {
		t.Errorf("unexpected PathPrefix: %s", parsed.PathPrefix)
	}
}
