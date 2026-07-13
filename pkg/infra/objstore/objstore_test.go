package objstore

import "testing"

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
