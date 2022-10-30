# Postbox 📬
A small library for constructing RFC 2822 style multipart messages. This library could be used to interact with a SMTP server to send mail.

```go
package main

import (
	"strings"

	"github.com/jeroenrinzema/postbox"
)

func main() {
	body := postbox.Part{
		ContentType: "message",
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
	}
}
```
