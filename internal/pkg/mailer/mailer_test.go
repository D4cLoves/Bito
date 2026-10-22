package mailer

import (
	"bufio"
	"context"
	"encoding/base64"
	"net"
	"strings"
	"testing"
	"time"
)

type fakeSMTP struct {
	addr string
	auth chan string
	data chan string
}

func startFakeSMTP(t *testing.T) *fakeSMTP {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	f := &fakeSMTP{addr: ln.Addr().String(), auth: make(chan string, 1), data: make(chan string, 1)}
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		write := func(s string) { _, _ = conn.Write([]byte(s + "\r\n")) }
		write("220 fake ESMTP")
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			cmd := strings.TrimSpace(line)
			switch {
			case strings.HasPrefix(cmd, "EHLO"):
				write("250-fake")
				write("250 AUTH PLAIN")
			case strings.HasPrefix(cmd, "AUTH PLAIN "):
				raw, _ := base64.StdEncoding.DecodeString(strings.TrimPrefix(cmd, "AUTH PLAIN "))
				f.auth <- strings.ReplaceAll(string(raw), "\x00", "|")
				write("235 ok")
			case strings.HasPrefix(cmd, "MAIL"), strings.HasPrefix(cmd, "RCPT"):
				write("250 ok")
			case cmd == "DATA":
				write("354 go")
				var b strings.Builder
				for {
					l, err := r.ReadString('\n')
					if err != nil || l == ".\r\n" {
						break
					}
					b.WriteString(l)
				}
				f.data <- b.String()
				write("250 queued")
			case cmd == "QUIT":
				write("221 bye")
				return
			default:
				write("502 no")
			}
		}
	}()
	return f
}

func (f *fakeSMTP) mailer(user, password string) *Mailer {
	host, port, _ := net.SplitHostPort(f.addr)
	return &Mailer{host: host, port: port, from: "bito@example.com", user: user, password: password}
}

func TestSendEmail_LogsInWhenUserIsSet(t *testing.T) {
	f := startFakeSMTP(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := f.mailer("bito@example.com", "app-password").SendVerificationCode(ctx, "player@example.com", "123456"); err != nil {
		t.Fatalf("send: %v", err)
	}

	if got := <-f.auth; got != "|bito@example.com|app-password" {
		t.Fatalf("auth = %q", got)
	}
	data := <-f.data
	for _, want := range []string{"From: Bito <bito@example.com>", "To: player@example.com", "Message-ID: <", "@example.com>", "123456"} {
		if !strings.Contains(data, want) {
			t.Fatalf("message lacks %q:\n%s", want, data)
		}
	}
}

func TestSendEmail_SkipsAuthWithoutUser(t *testing.T) {
	f := startFakeSMTP(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := f.mailer("", "").SendEmail(ctx, "player@example.com", "hi", "<p>hi</p>"); err != nil {
		t.Fatalf("send: %v", err)
	}

	select {
	case got := <-f.auth:
		t.Fatalf("unexpected auth %q", got)
	default:
	}
	if data := <-f.data; !strings.Contains(data, "<p>hi</p>") {
		t.Fatalf("body missing:\n%s", data)
	}
}
