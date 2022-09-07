// Package gomailer provides a simple email interface to integrate third party email services
package gomailer

import (
	"bytes"
	b64 "encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"path/filepath"
	"time"
)

const (
	defaultTimeout = 60 * time.Second

	_ driver = iota
	// MAILGUN driver
	MAILGUN
	// SENDGRID driver
	SENDGRID
	// POSTMARK driver
	POSTMARK
	// MAILJET driver
	MAILJET
	// CUSTOMERIO driver
	CUSTOMERIO
)

type (
	// mapData represents custom data type for mailer
	mapData map[string]interface{}
	// address describes an email address
	address struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Type  string `json:"type,omitempty"`
	}

	// attachment describes an email attachment
	attachment struct {
		Content     string `json:"content"`
		Type        string `json:"type"`
		FileName    string `json:"filename"`
		ContentID   string `json:"content_id"`
		Disposition string `json:"disposition,omitempty"`
		Size        int64  `json:"-"`
	}

	// Configs represents the configurations
	Configs struct {
		ServerToken    string        // ServerToken for service like postmarkapp
		AccountToken   string        // AccountToken for service like postmarkapp
		APIKey         string        // APIKey represents the API key for mail service like mailgun
		PrivateKey     string        // PrivateKey represents  the PrivateKey provided by service like mailjet
		PublicKey      string        // PublicKey represents  the PublicKey provided by service like mailjet
		BaseURL        string        // BaseURL represents the base url for service
		Domain         string        // Domain represents the domain of the service
		Username       string        // Username represents the username for service
		Password       string        // Password represents the password for service
		RequestTimeout time.Duration // RequestTimeout represents the timeout for http client call
	}

	// represents the driver type
	driver int

	// Mailer describes a common mailer interface for different email service
	Mailer interface {
		// From set sender email address for an email
		From(name, from string) Mailer
		// From set receipents email address for an email
		To(name, to string) Mailer
		// From set Cc receipents email address for an email
		Cc(name, to string) Mailer
		// From set Bcc receipents email address for an email
		Bcc(name, to string) Mailer
		// ReplyTo sets reply-to for an email
		ReplyTo(name, email string) Mailer
		// Subject sents Subject of an email
		Subject(sub string) Mailer
		// BodyHTML sets html body for an email
		BodyHTML(html string) Mailer
		// BodyText sets plain text body for an email
		BodyText(text string) Mailer
		// AttachmentFile sets email attachments from file name on disk
		AttachmentFile(file string) Mailer
		// AttachmentInlineFile sets email inline attachments from file name on disk
		AttachmentInlineFile(file string) Mailer
		// AttachmentReader sets email attachments from file name and reader
		AttachmentReader(file string, r io.Reader) Mailer
		// AttachmentInlineFile sets email inline attachments from file name and reader
		AttachmentInlineReader(file string, r io.Reader) Mailer
		// Send process an email sending
		Send() error
	}
)

// format return a formatted email string
func (a address) format() string {
	return fmt.Sprintf("%s <%s>", a.Name, a.Email)
}

// ReadFromFile read from valid file path
// encode the file into base64
func (a *attachment) ReadFromFile(path string) error {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	// encode file into base64
	sEnc := b64.StdEncoding.EncodeToString(b)
	//extract mime type from file
	ext := filepath.Ext(path)
	mType := mime.TypeByExtension(ext)
	// get file name
	fileName := filepath.Base(path)
	// modify the receiver
	a.Type = mType
	a.Content = sEnc
	a.FileName = fileName
	a.ContentID = fileName
	return nil
}

// ReadFromFile read from valid file path
// encode the file into base64
func (a *attachment) ReadFromReader(f string, sri readerInfo) error {
	b := &bytes.Buffer{}
	size, err := io.Copy(b, sri.r)
	if err != nil {
		return err
	}
	a.Size = size
	// encode file into base64
	sEnc := b64.StdEncoding.EncodeToString(b.Bytes())
	//extract mime type from file
	ext := filepath.Ext(f)
	mType := mime.TypeByExtension(ext)
	// get file name
	fileName := filepath.Base(f)
	// modify the receiver
	a.Type = mType
	a.Content = sEnc
	a.FileName = fileName
	a.ContentID = fileName
	return nil
}

// New Return a new mail driver
func New(d driver, c Configs) (Mailer, error) {
	return mailFactory(d, c)
}

// mailFactory return an email type depending on driver
func mailFactory(d driver, c Configs) (Mailer, error) {
	switch d {
	case MAILGUN:
		return &mailgun{
			configs: c,
			c: client{
				timeOut: c.RequestTimeout,
			},
		}, nil

	case MAILJET:
		return &mailjet{
			configs: c,
			c: client{
				timeOut: c.RequestTimeout,
			},
		}, nil

	case SENDGRID:
		return &sendgrid{
			configs: c,
			c: client{
				timeOut: c.RequestTimeout,
			},
		}, nil

	case POSTMARK:
		return &postmark{
			configs: c,
			c: client{
				timeOut: c.RequestTimeout,
			},
		}, nil

	case CUSTOMERIO:
		return &customerio{
			configs: c,
			c: client{
				timeOut: c.RequestTimeout,
			},
		}, nil

	default:
		return nil, errors.New("gomailer: unsupported mail driver")
	}
}
