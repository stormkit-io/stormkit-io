package shttp

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClientIP_XForwardedFor(t *testing.T) {
	req := &http.Request{
		Header:     make(http.Header),
		RemoteAddr: "192.168.1.1:1234",
	}
	req.Header.Set("X-Forwarded-For", "203.0.113.195")

	ip := ClientIP(req)
	assert.Equal(t, "203.0.113.195", ip)
}

func TestClientIP_XForwardedFor_Multiple(t *testing.T) {
	req := &http.Request{
		Header:     make(http.Header),
		RemoteAddr: "192.168.1.1:1234",
	}
	req.Header.Set("X-Forwarded-For", "203.0.113.195, 70.41.3.18, 150.172.238.178")

	// Should return the first IP (original client)
	ip := ClientIP(req)
	assert.Equal(t, "203.0.113.195", ip)
}

func TestClientIP_XRealIP(t *testing.T) {
	req := &http.Request{
		Header:     make(http.Header),
		RemoteAddr: "192.168.1.1:1234",
	}
	req.Header.Set("X-Real-IP", "203.0.113.195")

	ip := ClientIP(req)
	assert.Equal(t, "203.0.113.195", ip)
}

func TestClientIP_XForwardedFor_Priority(t *testing.T) {
	// X-Forwarded-For should take priority over X-Real-IP
	req := &http.Request{
		Header:     make(http.Header),
		RemoteAddr: "192.168.1.1:1234",
	}
	req.Header.Set("X-Forwarded-For", "203.0.113.195")
	req.Header.Set("X-Real-IP", "198.51.100.178")

	ip := ClientIP(req)
	assert.Equal(t, "203.0.113.195", ip)
}

func TestClientIP_RemoteAddr_IPv4(t *testing.T) {
	req := &http.Request{
		Header:     http.Header{},
		RemoteAddr: "203.0.113.195:52301",
	}

	ip := ClientIP(req)
	assert.Equal(t, "203.0.113.195", ip)
}

func TestClientIP_RemoteAddr_IPv6(t *testing.T) {
	req := &http.Request{
		Header:     http.Header{},
		RemoteAddr: "[2001:db8::1]:8080",
	}

	ip := ClientIP(req)
	assert.Equal(t, "2001:db8::1", ip)
}

func TestClientIP_RemoteAddr_NoPort(t *testing.T) {
	req := &http.Request{
		Header:     http.Header{},
		RemoteAddr: "203.0.113.195",
	}

	ip := ClientIP(req)
	assert.Equal(t, "203.0.113.195", ip)
}

func TestClientIP_Cloudflare(t *testing.T) {
	// Simulate Cloudflare proxy headers
	req := &http.Request{
		Header:     make(http.Header),
		RemoteAddr: "104.21.0.1:443",
	}
	req.Header.Set("X-Forwarded-For", "203.0.113.195")
	req.Header.Set("CF-Connecting-IP", "203.0.113.195")

	ip := ClientIP(req)
	assert.Equal(t, "203.0.113.195", ip)
}

func TestClientIP_Nginx(t *testing.T) {
	// Simulate Nginx proxy_pass headers
	req := &http.Request{
		Header:     make(http.Header),
		RemoteAddr: "10.0.0.1:8080",
	}
	req.Header.Set("X-Real-IP", "203.0.113.195")
	req.Header.Set("X-Forwarded-For", "203.0.113.195")

	ip := ClientIP(req)
	assert.Equal(t, "203.0.113.195", ip)
}

func TestClientIP_WithSpaces(t *testing.T) {
	req := &http.Request{
		Header:     make(http.Header),
		RemoteAddr: "192.168.1.1:1234",
	}
	req.Header.Set("X-Forwarded-For", "  203.0.113.195  ")

	ip := ClientIP(req)
	assert.Equal(t, "203.0.113.195", ip)
}
