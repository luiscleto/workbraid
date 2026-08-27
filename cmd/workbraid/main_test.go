package main

import "testing"

func TestOriginForLoopbackAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
		want    string
		ok      bool
	}{
		{name: "ipv4", address: "127.0.0.1:7788", want: "http://127.0.0.1:7788", ok: true},
		{name: "ipv6", address: "[::1]:7788", want: "http://[::1]:7788", ok: true},
		{name: "hostname rejected", address: "localhost:7788"},
		{name: "unspecified rejected", address: "0.0.0.0:7788"},
		{name: "non-loopback rejected", address: "192.0.2.1:7788"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := originForLoopbackAddress(test.address)
			if test.ok {
				if err != nil || got != test.want {
					t.Fatalf("originForLoopbackAddress(%q) = %q, %v; want %q", test.address, got, err, test.want)
				}
				return
			}
			if err == nil {
				t.Fatalf("originForLoopbackAddress(%q) unexpectedly succeeded with %q", test.address, got)
			}
		})
	}
}
