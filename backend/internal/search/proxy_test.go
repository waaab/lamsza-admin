package search

import (
	"net"
	"testing"
)

func TestParsePublicProxyURL(t *testing.T) {
	u, err := parsePublicProxyURL("https://Example.com:443/a/b?x=1#frag")
	if err != nil {
		t.Fatal(err)
	}
	if u.String() != "https://example.com/a/b?x=1" {
		t.Fatalf("canonical %s", u.String())
	}
	if _, err := parsePublicProxyURL("file:///etc/passwd"); err == nil {
		t.Fatal("file scheme should fail")
	}
	if _, err := parsePublicProxyURL("https://user:pass@example.com/a"); err == nil {
		t.Fatal("userinfo should fail")
	}
}

func TestUpgradePhotoURL(t *testing.T) {
	thumb := "https://upload.wikimedia.org/wikipedia/commons/thumb/a/ab/Foo.jpg/320px-Foo.jpg"
	got := upgradePhotoURL(thumb)
	want := "https://upload.wikimedia.org/wikipedia/commons/a/ab/Foo.jpg"
	if got != want {
		t.Fatalf("upgrade %s", got)
	}
}

func TestPublicIP(t *testing.T) {
	if publicIP(net.ParseIP("127.0.0.1")) || publicIP(net.ParseIP("10.1.2.3")) || publicIP(net.ParseIP("169.254.169.254")) || publicIP(net.ParseIP("::1")) {
		t.Fatal("private addresses should be blocked")
	}
	if !publicIP(net.ParseIP("93.184.216.34")) {
		t.Fatal("public address should pass")
	}
}
