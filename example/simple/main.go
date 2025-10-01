package main

import (
	"io"
	"os"
	"strings"

	"github.com/jeroenrinzema/postbox"
)

func main() {
	body := postbox.Part{
		ContentType: "text/plain",
		Encoding:    postbox.Base64,
		Reader:      strings.NewReader("https://www.youtube.com/watch?v=dQw4w9WgXcQ"),
	}

	mail := postbox.Envelope{
		From:    "john@example.com",
		Sender:  "john@example.com",
		ReplyTo: "reply@example.com",
		To:      []string{"bil@example.com", "dan@example.com"},
		Subject: "Check this out!",
		Parts:   []*postbox.Part{&body},
		Charset: "utf-8",
	}

	mail.Write(nopCloser{os.Stdout})
}

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error { return nil }
