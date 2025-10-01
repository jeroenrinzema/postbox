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
func (h Headers) Write(writer io.Writer) (err error) {
	for property, values := range h {
		_, err = writer.Write([]byte(property))
		if err != nil {
			return err
		}

		if len(values) == 0 {
			_, err = writer.Write([]byte(":" + CRLF))
			if err != nil {
				return err
			}
			continue
		}

		_, err = writer.Write([]byte(": "))
		if err != nil {
			return err
		}

		values := strings.Join(values, "; ")
		reader := strings.NewReader(values)

		_, err = io.Copy(writer, reader)
		if err != nil {
			return err
		}

		_, err = writer.Write([]byte(CRLF))
		if err != nil {
			return err
		}
	}

	return nil
}

// Part represents a multiform part
type Part struct {
	ContentType string
	Encoding    Encoding
	Reader      io.Reader
}

// Write writes the part to the given io writer
func (p *Part) Write(writer io.Writer, charset string) (err error) {
	headers := Headers{
		"Content-Type":              {p.ContentType, "charset=" + charset},
		"Content-Transfer-Encoding": {string(p.Encoding)},
	}

	err = headers.Write(writer)
	if err != nil {
		return err
	}

	_, err = writer.Write([]byte(CRLF))
	if err != nil {
		return err
	}

	switch p.Encoding {
	case QuotedPrintable:
		reader := quotedprintable.NewReader(p.Reader)
		_, err = io.Copy(writer, reader)
		if err != nil {
			return err
		}
	case Base64:
		encoder := base64.NewEncoder(base64.StdEncoding, writer)
		_, err = io.Copy(encoder, p.Reader)
		if err != nil {
			return err
		}

		err = encoder.Close()
		if err != nil {
			return err
		}
	default:
		_, err = io.Copy(writer, p.Reader)
		if err != nil {
			return err
		}
	}

	_, err = writer.Write([]byte(CRLF))
	return err
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
func (b *Boundary) Mark() (err error) {
	_, err = b.writer.Write([]byte("--" + b.Identifier + CRLF))
	return err
}

// End marks the boundary as ended
func (b *Boundary) End() (err error) {
	_, err = b.writer.Write([]byte("--" + b.Identifier + "--" + CRLF + CRLF))
	return err
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
func (e *Envelope) Write(writer io.WriteCloser) (err error) {
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

	err = headers.Write(writer)
	if err != nil {
		return err
	}

	mixed := NewBoundary(writer, "multipart/mixed")
	err = mixed.Mark()
	if err != nil {
		return err
	}

	related := NewBoundary(writer, "multipart/related")
	err = related.Mark()
	if err != nil {
		return err
	}

	alternative := NewBoundary(writer, "multipart/alternative")

	for _, part := range e.Parts {
		err = alternative.Mark()
		if err != nil {
			return err
		}

		err = part.Write(writer, e.Charset)
		if err != nil {
			return err
		}
	}

	err = alternative.End()
	if err != nil {
		return err
	}

	err = related.End()
	if err != nil {
		return err
	}

	err = mixed.End()
	if err != nil {
		return err
	}

	return writer.Close()
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
