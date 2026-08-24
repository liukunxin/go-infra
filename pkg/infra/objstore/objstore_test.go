package objstore

import (
	"strings"
	"testing"
)

func TestConfigNormalize(t *testing.T) {
	cfg := &Config{
		Endpoint: "  https://ks3-cn-beijing.ksyun.com/ ",
		Bucket:   " uf-docer ",
	}
	cfg.Normalize()
	if cfg.Endpoint != "ks3-cn-beijing.ksyun.com" {
		t.Fatalf("endpoint=%q", cfg.Endpoint)
	}
	if cfg.Bucket != "uf-docer" {
		t.Fatalf("bucket=%q", cfg.Bucket)
	}
	if cfg.Region != DefaultRegion {
		t.Fatalf("region=%q", cfg.Region)
	}
}

func TestServiceBaseEndpoint(t *testing.T) {
	if got := serviceBaseEndpoint("ks3-cn-beijing.ksyun.com"); got != "https://ks3-cn-beijing.ksyun.com" {
		t.Fatalf("got %q", got)
	}
}

func TestNormalizeObjectACL(t *testing.T) {
	if got := NormalizeObjectACL("public-read"); got != string(ObjectACLPublicRead) {
		t.Fatalf("got %q", got)
	}
	if !IsPublicObjectACL("public-read") {
		t.Fatal("public-read should be public")
	}
	if IsPublicObjectACL("private") {
		t.Fatal("private should not be public")
	}
}

func TestBuildObjectURL(t *testing.T) {
	tests := []struct {
		name         string
		endpoint     string
		bucket       string
		key          string
		usePathStyle bool
		want         string
	}{
		{
			name:     "virtual-hosted",
			endpoint: "ks3-cn-beijing.ksyun.com",
			bucket:   "uf-docer",
			key:      "wps_study_assistant/a b.pdf",
			want:     "https://uf-docer.ks3-cn-beijing.ksyun.com/wps_study_assistant/a%20b.pdf",
		},
		{
			name:         "path-style",
			endpoint:     "ks3-cn-beijing.ksyun.com",
			bucket:       "uf-docer",
			key:          "dir/file.pdf",
			usePathStyle: true,
			want:         "https://ks3-cn-beijing.ksyun.com/uf-docer/dir/file.pdf",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildObjectURL(tc.endpoint, tc.bucket, tc.key, tc.usePathStyle)
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestObjectURLClient(t *testing.T) {
	c := &Client{
		bucket:       "uf-docer",
		endpoint:     "ks3-cn-beijing.ksyun.com",
		usePathStyle: false,
	}
	got := c.ObjectURL("", "wps_study_assistant/a.pdf")
	want := "https://uf-docer.ks3-cn-beijing.ksyun.com/wps_study_assistant/a.pdf"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestPublicURLClient(t *testing.T) {
	c := &Client{
		bucket:        "uf-docer",
		endpoint:      "ks3-cn-beijing.ksyun.com",
		publicBaseURL: "https://cdn.example.com",
	}
	got := c.PublicURL("", "wps_study_assistant/a b.pdf")
	want := "https://cdn.example.com/wps_study_assistant/a%20b.pdf"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}

	c.publicBaseURL = ""
	got = c.PublicURL("", "wps_study_assistant/a.pdf")
	want = "https://uf-docer.ks3-cn-beijing.ksyun.com/wps_study_assistant/a.pdf"
	if got != want {
		t.Fatalf("fallback got %q want %q", got, want)
	}

	got = c.PublicURL("", "wps_study_assistant/a b.pdf")
	want = "https://uf-docer.ks3-cn-beijing.ksyun.com/wps_study_assistant/a%20b.pdf"
	if got != want {
		t.Fatalf("fallback with space got %q want %q", got, want)
	}
	if strings.Contains(got, "%2520") {
		t.Fatalf("double-encoded space in fallback URL: %q", got)
	}
}

func TestBuildPublicURLNoDoubleEncode(t *testing.T) {
	key := "wps_study_assistant/tmp/source_upload/1666508657790/2091856958502998016/WPS Slides Quick Start Guide(1).pdf"
	got := buildPublicURL(
		"",
		"s3.us-west-2.amazonaws.com",
		"2c-assist-svc-us-prod-study-assist",
		key,
		false,
	)
	want := "https://2c-assist-svc-us-prod-study-assist.s3.us-west-2.amazonaws.com/wps_study_assistant/tmp/source_upload/1666508657790/2091856958502998016/WPS%20Slides%20Quick%20Start%20Guide%281%29.pdf"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if strings.Contains(got, "%2520") || strings.Contains(got, "%2528") || strings.Contains(got, "%2529") {
		t.Fatalf("double-encoded URL: %q", got)
	}
}

// PublicURL 在未配置 public_base_url 时应与 ObjectURL 完全一致（单次编码）。
// 修复前 fallback 会双重 PathEscape，仅「需编码字符」的文件名会出错；纯字母数字文件名碰巧相同。
func TestPublicURLFallbackMatchesObjectURL(t *testing.T) {
	const (
		endpoint = "s3.us-west-2.amazonaws.com"
		bucket   = "my-bucket"
	)
	fileNames := []string{
		"lecture.pdf",                              // 常见成功 case：无需编码
		"simple-file_v1.0.pdf",                     // 字母数字 + .-_ 
		"a b.pdf",                                  // 空格
		"WPS Slides Quick Start Guide(1).pdf",      // 空格 + 括号（线上失败 case）
		"100%complete.pdf",                         // 文件名含 %
		"notes#draft.pdf",                          // #
		"中文 课件.pdf",                                // 中文 + 空格
		"file+plus.pdf",                            // +
	}
	for _, name := range fileNames {
		t.Run(name, func(t *testing.T) {
			key := "wps_study_assistant/tmp/source_upload/10001/up1/" + name
			public := buildPublicURL("", endpoint, bucket, key, false)
			object := buildObjectURL(endpoint, bucket, key, false)
			if public != object {
				t.Fatalf("PublicURL fallback must match ObjectURL\npublic=%q\nobject=%q", public, object)
			}
		})
	}
}

func TestBuildPublicURLWithCDNBase(t *testing.T) {
	key := "wps_study_assistant/a b.pdf"
	got := buildPublicURL("https://cdn.example.com/", "ignored", "ignored", key, false)
	want := "https://cdn.example.com/wps_study_assistant/a%20b.pdf"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if strings.Contains(got, "%2520") {
		t.Fatalf("CDN base must not double-encode: %q", got)
	}
}

func TestBuildPublicURLPathStyleFallback(t *testing.T) {
	key := "dir/a b.pdf"
	got := buildPublicURL("", "s3.amazonaws.com", "bucket", key, true)
	want := buildObjectURL("s3.amazonaws.com", "bucket", key, true)
	if got != want {
		t.Fatalf("path-style fallback got %q want %q", got, want)
	}
}

func TestJoinKey(t *testing.T) {
	if got := JoinKey("wps_study_assistant", "images", "a.png"); got != "wps_study_assistant/images/a.png" {
		t.Fatalf("got %q", got)
	}
	if got := JoinKey("", "images", "a.png"); got != "images/a.png" {
		t.Fatalf("got %q", got)
	}
}

func TestConfigObjectKey(t *testing.T) {
	cfg := &Config{KeyPrefix: "wps_study_assistant"}
	if got := cfg.ObjectKey("source_upload", "u1", "f.pdf"); got != "wps_study_assistant/source_upload/u1/f.pdf" {
		t.Fatalf("got %q", got)
	}
}

func TestValidateObjectKey(t *testing.T) {
	if err := validateObjectKey(""); err == nil {
		t.Fatal("expected error for empty key")
	}
	if err := validateObjectKey("a.txt"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPresignPutRequiredHeaders(t *testing.T) {
	headers := presignPutRequiredHeaders(PresignPutOptions{
		ACL:         "public-read",
		ContentType: "application/pdf",
	})
	if headers[ObjectACLHeader] != string(ObjectACLPublicRead) {
		t.Fatalf("acl header=%v", headers)
	}
	if headers["Content-Type"] != "application/pdf" {
		t.Fatalf("content-type header=%v", headers)
	}
}
