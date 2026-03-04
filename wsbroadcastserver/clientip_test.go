// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package wsbroadcastserver

import (
	"net"
	"net/textproto"
	"testing"
)

func TestResolveClientIP_SingleHeader(t *testing.T) {
	headers := map[string]string{
		textproto.CanonicalMIMEHeaderKey("CF-Connecting-IP"): "1.2.3.4",
	}
	ip := resolveClientIP(headers, []string{"CF-Connecting-IP"}, []int{0})
	if !ip.Equal(net.ParseIP("1.2.3.4")) {
		t.Fatalf("expected 1.2.3.4, got %v", ip)
	}
}

func TestResolveClientIP_CommaSeparatedRightmost(t *testing.T) {
	headers := map[string]string{
		textproto.CanonicalMIMEHeaderKey("X-Forwarded-For"): "1.2.3.4, 2.3.4.5, 3.4.5.6",
	}
	// nth=0 means rightmost
	ip := resolveClientIP(headers, []string{"X-Forwarded-For"}, []int{0})
	if !ip.Equal(net.ParseIP("3.4.5.6")) {
		t.Fatalf("expected 3.4.5.6 (rightmost), got %v", ip)
	}
}

func TestResolveClientIP_NthFromRight(t *testing.T) {
	headers := map[string]string{
		textproto.CanonicalMIMEHeaderKey("X-Forwarded-For"): "1.2.3.4, 2.3.4.5, 3.4.5.6",
	}
	// nth=1 means second from right
	ip := resolveClientIP(headers, []string{"X-Forwarded-For"}, []int{1})
	if !ip.Equal(net.ParseIP("2.3.4.5")) {
		t.Fatalf("expected 2.3.4.5 (second from right), got %v", ip)
	}

	// nth=2 means third from right (leftmost in this case)
	ip = resolveClientIP(headers, []string{"X-Forwarded-For"}, []int{2})
	if !ip.Equal(net.ParseIP("1.2.3.4")) {
		t.Fatalf("expected 1.2.3.4 (leftmost), got %v", ip)
	}
}

func TestResolveClientIP_NthOutOfRange(t *testing.T) {
	headers := map[string]string{
		textproto.CanonicalMIMEHeaderKey("X-Forwarded-For"): "1.2.3.4",
	}
	// nth=5 is out of range for a single-element list
	ip := resolveClientIP(headers, []string{"X-Forwarded-For"}, []int{5})
	if ip != nil {
		t.Fatalf("expected nil for out-of-range nth, got %v", ip)
	}
}

func TestResolveClientIP_PriorityOrder(t *testing.T) {
	headers := map[string]string{
		textproto.CanonicalMIMEHeaderKey("CF-Connecting-IP"): "10.0.0.1",
		textproto.CanonicalMIMEHeaderKey("X-Forwarded-For"):  "20.0.0.1, 30.0.0.1",
	}
	// CF-Connecting-IP is first in priority, should be used
	ip := resolveClientIP(headers, []string{"CF-Connecting-IP", "X-Forwarded-For"}, []int{0, 0})
	if !ip.Equal(net.ParseIP("10.0.0.1")) {
		t.Fatalf("expected 10.0.0.1 from higher-priority header, got %v", ip)
	}
}

func TestResolveClientIP_FallbackToSecondHeader(t *testing.T) {
	headers := map[string]string{
		textproto.CanonicalMIMEHeaderKey("X-Forwarded-For"): "20.0.0.1, 30.0.0.1",
	}
	// CF-Connecting-IP is missing, should fall back to X-Forwarded-For
	ip := resolveClientIP(headers, []string{"CF-Connecting-IP", "X-Forwarded-For"}, []int{0, 0})
	if !ip.Equal(net.ParseIP("30.0.0.1")) {
		t.Fatalf("expected 30.0.0.1 from fallback header, got %v", ip)
	}
}

func TestResolveClientIP_NoHeaders(t *testing.T) {
	headers := map[string]string{}
	ip := resolveClientIP(headers, []string{"CF-Connecting-IP"}, []int{0})
	if ip != nil {
		t.Fatalf("expected nil when no headers present, got %v", ip)
	}
}

func TestResolveClientIP_EmptyConfig(t *testing.T) {
	headers := map[string]string{
		textproto.CanonicalMIMEHeaderKey("CF-Connecting-IP"): "1.2.3.4",
	}
	ip := resolveClientIP(headers, []string{}, []int{})
	if ip != nil {
		t.Fatalf("expected nil with empty config, got %v", ip)
	}
}

func TestResolveClientIP_MissingNthElement(t *testing.T) {
	headers := map[string]string{
		textproto.CanonicalMIMEHeaderKey("X-Forwarded-For"): "1.2.3.4, 2.3.4.5",
	}
	// nthElements slice is shorter than headers slice; should default to 0 (rightmost)
	ip := resolveClientIP(headers, []string{"X-Forwarded-For"}, []int{})
	if !ip.Equal(net.ParseIP("2.3.4.5")) {
		t.Fatalf("expected 2.3.4.5 (rightmost, default nth=0), got %v", ip)
	}
}

func TestResolveClientIP_InvalidIP(t *testing.T) {
	headers := map[string]string{
		textproto.CanonicalMIMEHeaderKey("X-Forwarded-For"): "not-an-ip",
	}
	ip := resolveClientIP(headers, []string{"X-Forwarded-For"}, []int{0})
	if ip != nil {
		t.Fatalf("expected nil for invalid IP, got %v", ip)
	}
}

func TestResolveClientIP_InvalidIPFallsThrough(t *testing.T) {
	headers := map[string]string{
		textproto.CanonicalMIMEHeaderKey("X-Real-Ip"):      "not-an-ip",
		textproto.CanonicalMIMEHeaderKey("X-Forwarded-For"): "5.6.7.8",
	}
	// First header has invalid IP, should fall through to second
	ip := resolveClientIP(headers, []string{"X-Real-Ip", "X-Forwarded-For"}, []int{0, 0})
	if !ip.Equal(net.ParseIP("5.6.7.8")) {
		t.Fatalf("expected 5.6.7.8 after invalid first header, got %v", ip)
	}
}

func TestResolveClientIP_IPv6(t *testing.T) {
	headers := map[string]string{
		textproto.CanonicalMIMEHeaderKey("CF-Connecting-IP"): "2001:db8::1",
	}
	ip := resolveClientIP(headers, []string{"CF-Connecting-IP"}, []int{0})
	if !ip.Equal(net.ParseIP("2001:db8::1")) {
		t.Fatalf("expected 2001:db8::1, got %v", ip)
	}
}

func TestResolveClientIP_CaseInsensitiveHeader(t *testing.T) {
	headers := map[string]string{
		textproto.CanonicalMIMEHeaderKey("x-forwarded-for"): "1.2.3.4",
	}
	// Config uses different casing
	ip := resolveClientIP(headers, []string{"X-FORWARDED-FOR"}, []int{0})
	if !ip.Equal(net.ParseIP("1.2.3.4")) {
		t.Fatalf("expected case-insensitive match, got %v", ip)
	}
}
