package postbox

import (
	"io"
	"strings"
	"testing"
	"time"
)

// TestWritingHeaders test if the correct headers are written to the given io.Writer
func TestWritingHeaders(t *testing.T) {
	headers := []string{
		"From: john@example.com",
		"To: john@example.com",
		"Reply-To: john@example.com",
		"Mime-Version: 1.0",
		"Date: Tue, 10 Nov 2009 23:00:00 +0100",
		"Cc: john@example.com; boss@example.com",
		"Subject: hello world",
	}

	loc, _ := time.LoadLocation("Europe/Amsterdam")
	envelope := Envelope{
		Date:    time.Date(2009, 11, 10, 23, 0, 0, 0, loc),
		From:    "john@example.com",
		Sender:  "john@example.com",
		ReplyTo: "john@example.com",
		To:      []string{"john@example.com"},
		Cc:      []string{"john@example.com", "boss@example.com"},
		Subject: "hello world",
		Charset: "UTF-8",
	}

	reader, writer := io.Pipe()
	go envelope.Write(writer)

	last := ""
	line := ""
	buffer := make([]byte, 1)

reader:
	for {
		n, err := reader.Read(buffer)
		if err != nil && err != io.EOF {
			t.Fatal(err)
		}

		str := string(buffer[:n])
		if str == LF && last == CR {
			for index, header := range headers {
				if header == line[:len(line)-1] {
					line = ""
					headers = append(headers[:index], headers[index+1:]...)
					continue reader
				}
			}

			if len(headers) != 0 {
				t.Fatal("Unexpected header:", line)
			}
		}

		line += str

		if err == io.EOF {
			break
		}

		last = str
	}
}

// TestWritingContent test if the correct headers are written to the given io.Writer
func TestWritingContent(t *testing.T) {
	plain := "hello world"
	html := "<p>hello <b>world</b></p>"

	expected := []string{
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: UTF-8",
		plain,
		"Content-Type: text/html; charset=UTF-8",
		"Content-Transfer-Encoding: UTF-8",
		html,
	}

	envelope := Envelope{
		Charset: "UTF-8",
		Parts: []*Part{
			{
				ContentType: "text/plain",
				Encoding:    "UTF-8",
				Reader:      strings.NewReader(plain),
			},
			{
				ContentType: "text/html",
				Encoding:    "UTF-8",
				Reader:      strings.NewReader(html),
			},
		},
	}

	reader, writer := io.Pipe()
	go envelope.Write(writer)

	last := ""
	line := ""
	buffer := make([]byte, 1)

	for {
		n, err := reader.Read(buffer)
		if err != nil && err != io.EOF {
			t.Fatal(err)
		}

		str := string(buffer[:n])
		if str == LF && last == CR {
			compare := line[:len(line)-1]

			for index, value := range expected {
				if value == compare {
					expected = append(expected[:index], expected[index+1:]...)
					break
				}
			}

			line = ""
		} else {
			line += str
		}

		if err == io.EOF {
			break
		}

		last = str
	}

	if len(expected) != 0 {
		t.Fatal("Not all expectations were met:", expected)
	}
}
