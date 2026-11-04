//go:build android && !cgo

package main

// Without cgo, Go's resolver on Android has no /etc/resolv.conf and falls
// back to 127.0.0.1:53, so every DNS lookup fails. Builds made without the
// Android NDK (CGO_ENABLED=0) therefore point the Go resolver at the
// nameservers from Termux's resolv.conf, or at public DNS if that is missing.

import (
	"bufio"
	"context"
	"net"
	"os"
	"strings"
	"time"
)

func termuxNameservers() []string {
	paths := []string{"/etc/resolv.conf"}
	if prefix := os.Getenv("PREFIX"); prefix != "" {
		paths = append([]string{prefix + "/etc/resolv.conf"}, paths...)
	}
	paths = append(paths, "/data/data/com.termux/files/usr/etc/resolv.conf")
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		var out []string
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			fields := strings.Fields(sc.Text())
			if len(fields) >= 2 && fields[0] == "nameserver" && net.ParseIP(fields[1]) != nil {
				out = append(out, net.JoinHostPort(fields[1], "53"))
			}
		}
		f.Close()
		if len(out) > 0 {
			return out
		}
	}
	return []string{"8.8.8.8:53", "1.1.1.1:53"}
}

func init() {
	servers := termuxNameservers()
	d := net.Dialer{Timeout: 5 * time.Second}
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			var lastErr error
			for _, s := range servers {
				conn, err := d.DialContext(ctx, network, s)
				if err == nil {
					return conn, nil
				}
				lastErr = err
			}
			return nil, lastErr
		},
	}
}
