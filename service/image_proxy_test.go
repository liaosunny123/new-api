package service

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/setting/model_setting"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

func newTestContext(host string) *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/images/generations", nil)
	c.Request.Host = host
	return c
}

func TestImageProxyTokenRoundTrip(t *testing.T) {
	url := "https://files.example.com/a/b/c.png?x=1"
	token, err := EncodeImageProxyToken(url, 42)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if strings.ContainsAny(token, "/+= ") {
		t.Fatalf("token should be url-safe base64: %q", token)
	}
	gotURL, gotCh, err := ParseImageProxyToken(token)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if gotURL != url || gotCh != 42 {
		t.Fatalf("round trip mismatch: %q %d", gotURL, gotCh)
	}
	if _, _, err := ParseImageProxyToken("not-a-valid-token"); err == nil {
		t.Fatalf("expected error for garbage token")
	}
}

func TestBuildImageProxyURL(t *testing.T) {
	model_setting.GetGlobalSettings().ImageProxyEnabled = true
	defer func() { model_setting.GetGlobalSettings().ImageProxyEnabled = false }()

	c := newTestContext("ai.soruxgpt.com")

	// external URL -> proxied
	proxied, ok := BuildImageProxyURL(c, "https://files.example.com/x.png", 1)
	if !ok || !strings.HasPrefix(proxied, "http://ai.soruxgpt.com/file-assets/") {
		t.Fatalf("expected proxy url, got ok=%v url=%q", ok, proxied)
	}

	// already-local URL -> skip
	if _, ok := BuildImageProxyURL(c, "https://ai.soruxgpt.com/foo.png", 1); ok {
		t.Fatalf("local url should not be proxied")
	}
	// data uri / non-http -> skip
	if _, ok := BuildImageProxyURL(c, "data:image/png;base64,AAAA", 1); ok {
		t.Fatalf("data uri should not be proxied")
	}
}

func TestRewriteImageResponseBody(t *testing.T) {
	model_setting.GetGlobalSettings().ImageProxyEnabled = true
	defer func() { model_setting.GetGlobalSettings().ImageProxyEnabled = false }()

	c := newTestContext("ai.soruxgpt.com")
	body := []byte(`{"created":1781972469,"data":[{"url":"https://files.example.com/a.png"},{"b64_json":"AAAA"}],"usage":{"total_tokens":3358}}`)

	out := RewriteImageResponseBody(c, 7, body)

	newURL := gjson.GetBytes(out, "data.0.url").String()
	if !strings.HasPrefix(newURL, "http://ai.soruxgpt.com/file-assets/") {
		t.Fatalf("data.0.url not rewritten: %q", newURL)
	}
	// other fields preserved
	if gjson.GetBytes(out, "created").Int() != 1781972469 {
		t.Fatalf("created not preserved: %s", out)
	}
	if gjson.GetBytes(out, "usage.total_tokens").Int() != 3358 {
		t.Fatalf("usage not preserved: %s", out)
	}
	if gjson.GetBytes(out, "data.1.b64_json").String() != "AAAA" {
		t.Fatalf("b64_json not preserved: %s", out)
	}

	// the rewritten token must decode back to the original upstream URL
	token := strings.TrimPrefix(newURL, "http://ai.soruxgpt.com/file-assets/")
	gotURL, gotCh, err := ParseImageProxyToken(token)
	if err != nil || gotURL != "https://files.example.com/a.png" || gotCh != 7 {
		t.Fatalf("token decode mismatch: %q %d %v", gotURL, gotCh, err)
	}

	// disabled -> unchanged
	model_setting.GetGlobalSettings().ImageProxyEnabled = false
	if string(RewriteImageResponseBody(c, 7, body)) != string(body) {
		t.Fatalf("disabled should be no-op")
	}
}
