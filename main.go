package postbox

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime/quotedprintable"
	"strings"
	"time"
)

// Encoding represents a MIME encoding scheme like quoted-printable or base64.
type Encoding string

const (
	// QuotedPrintable represents the quoted-printable encoding as defined in
	// RFC 2045.
	QuotedPrintable Encoding = "quoted-printable"
	// Base64 represents the base64 encoding as defined in RFC 2045.
	Base64 Encoding = "base64"
	// Unencoded can be used to avoid encoding the body of an email. The headers
	// will still be encoded using quoted-printable encoding.
	Unencoded Encoding = "8bit"
)

// CR represents a ASCII CR
const CR = "\r"

// LF represents a ASCII LR
const LF = "\n"

// CRLF represents a ASCII CR+LF
const CRLF = CR + LF

// ContentType and it's boundry
type ContentType string

// Headers is a representation of a multiform part header
type Headers map[string][]string

// Write writes the headers to the given io.Writer
func (h Headers) Write(writer io.Writer) {
	for property, values := range h {
		writer.Write([]byte(property))

		if len(values) == 0 {
			writer.Write([]byte(":" + CRLF))
			continue
		}

		writer.Write([]byte(": "))

		values := strings.Join(values, "; ")
		reader := strings.NewReader(values)

		io.Copy(writer, reader)
		writer.Write([]byte(CRLF))
	}
}

// Part represents a multiform part
type Part struct {
	ContentType string
	Encoding    Encoding
	Reader      io.Reader
}

// Write writes the part to the given io writer
func (p *Part) Write(writer io.Writer, charset string) {
	headers := Headers{
		"Content-Type":              {p.ContentType, "charset=" + charset},
		"Content-Transfer-Encoding": {string(p.Encoding)},
	}

	headers.Write(writer)
	writer.Write([]byte(CRLF))

	switch p.Encoding {
	case QuotedPrintable:
		reader := quotedprintable.NewReader(p.Reader)
		io.Copy(writer, reader)
	case Base64:
		encoder := base64.NewEncoder(base64.StdEncoding, writer)
		io.Copy(encoder, p.Reader)
		encoder.Close()
	default:
		io.Copy(writer, p.Reader)
	}

	writer.Write([]byte(CRLF))
}

// File represents a multiform file
type File struct {
	Name     string
	Header   map[string][]string
	CopyFunc func(w io.Writer) error
}

// Boundary represents a multipart boundary
type Boundary struct {
	Identifier string
	writer     io.Writer
}

// NewBoundary starts a new multipart context and generates a new boundary.
// The headers are written to the given io.Writer.
func NewBoundary(writer io.Writer, mime string) Boundary {
	identifier := RandomBoundary()
	headers := Headers{
		"Content-Type": {mime, "boundary=" + identifier},
	}

	boundary := Boundary{
		Identifier: identifier,
		writer:     writer,
	}

	headers.Write(writer)
	writer.Write([]byte(CRLF))

	return boundary
}

// Mark appends the boundary identifier to the set io.Writer
func (b *Boundary) Mark() {
	b.writer.Write([]byte("--" + b.Identifier + CRLF))
}

// End marks the boundary as ended
func (b *Boundary) End() {
	b.writer.Write([]byte("--" + b.Identifier + "--" + CRLF + CRLF))
}

// Envelope is responsible for the generation of RFC 822-style emails.
// Specifications mentioned:
// - RFC 2822 - Internet Message Format
// - RFC 2387 - The MIME Multipart/Related Content-type
// - RFC 1341 - MIME  (Multipurpose Internet Mail Extensions)
// - RFC 4021 - Registration of Mail and MIME Header Fields
type Envelope struct {
	Date        time.Time // RFC 4021 2.1.1
	From        string    // RFC 4021 2.1.2
	Sender      string    // RFC 4021 2.1.3
	ReplyTo     string    // RFC 4021 2.1.4
	To          []string  // RFC 4021 2.1.5
	Cc          []string  // RFC 4021 2.1.6
	Subject     string    // RFC 4021 2.1.11
	Parts       []*Part   // RFC 1341 7.2
	Embedded    []*File   // RFC 2387
	Attachments []*File   // RFC 1341 7.2
	Charset     string
}

// Write writes the smtp message as multiform to the given io.Writer
func (e *Envelope) Write(writer io.WriteCloser) {
	if e.Date.IsZero() {
		e.Date = time.Now()
	}

	headers := Headers{
		"Date":         {e.Date.Format(time.RFC1123Z)},
		"From":         {e.From},
		"To":           e.To,
		"Cc":           e.Cc,
		"Reply-To":     {e.ReplyTo},
		"Subject":      {e.Subject},
		"Mime-Version": {"1.0"},
	}

	headers.Write(writer)

	mixed := NewBoundary(writer, "multipart/mixed")
	mixed.Mark()

	related := NewBoundary(writer, "multipart/related")
	related.Mark()

	alternative := NewBoundary(writer, "multipart/alternative")

	for _, part := range e.Parts {
		alternative.Mark()
		part.Write(writer, e.Charset)
	}

	alternative.End()
	related.End()
	mixed.End()
	writer.Close()
}

// RandomBoundary generates a new random boundary
func RandomBoundary() string {
	var buf [30]byte
	_, err := io.ReadFull(rand.Reader, buf[:])
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", buf[:])
}
