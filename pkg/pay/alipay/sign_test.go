package alipay

import (
	"net/url"
	"testing"
)

func TestBuildSignContent(t *testing.T) {
	v := url.Values{}
	v.Set("b", "2")
	v.Set("a", "1")
	v.Set("sign", "should_skip")
	v.Set("sign_type", "RSA2")
	v.Set("empty", "")
	got := buildSignContent(v)
	want := "a=1&b=2"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
