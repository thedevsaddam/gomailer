package gomailer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	cio "github.com/customerio/go-customerio/v3"
	"github.com/google/uuid"
)

const (
	// customerioMaxFileSize describes the max file size in bytes for sending per email for customerio
	customerioMaxFileSize int64 = 30 * 1000000
	// customerioMaxReceipents describes the max receipents per email
	customerioMaxReceipents = 1000
)

type (

	// customerio describes a customerio type
	customerio struct {
		c                       client
		configs                 Configs
		from                    address
		toList                  []address
		ccList                  []address
		bccList                 []address
		replyTo                 address
		subject                 string
		bodyHTML                string
		bodyText                string
		attachmentFiles         []string
		attachmentInlineFiles   []string
		attachmentReaders       map[string]readerInfo
		attachmentInlineReaders map[string]readerInfo
		attachments             []attachment
	}

	customerioContent struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	}
)

// From sets an email sender address
func (c *customerio) From(name, from string) Mailer {
	c.from = address{Name: name, Email: from}
	return c
}

// To sets receipents of an email
func (c *customerio) To(name, to string) Mailer {
	c.toList = append(c.toList, address{Name: name, Email: to})
	return c
}

// Cc sets Cc receipents of an email
func (c *customerio) Cc(name, to string) Mailer {
	c.ccList = append(c.ccList, address{Name: name, Email: to})
	return c
}

// Bcc sets Bcc receipents of an email
func (c *customerio) Bcc(name, to string) Mailer {
	c.bccList = append(c.bccList, address{Name: name, Email: to})
	return c
}

// ReplyTo sets the reply-to address of an email
func (c *customerio) ReplyTo(name, email string) Mailer {
	c.replyTo = address{Name: name, Email: email}
	return c
}

// Subject sets subject of an email
func (c *customerio) Subject(subject string) Mailer {
	c.subject = subject
	return c
}

// BodyHTML sets html body for an email
func (c *customerio) BodyHTML(body string) Mailer {
	c.bodyHTML = body
	return c
}

// BodyText sets plain text email body for an email
func (c *customerio) BodyText(body string) Mailer {
	c.bodyText = body
	return c
}

// AttachmentFile set email attachments
func (c *customerio) AttachmentFile(file string) Mailer {
	c.attachmentFiles = append(c.attachmentFiles, file)
	return c
}

// AttachmentInlineFile set email inline attachment
func (c *customerio) AttachmentInlineFile(file string) Mailer {
	c.attachmentInlineFiles = append(c.attachmentInlineFiles, file)
	return c
}

// AttachmentReader set email attachments
func (c *customerio) AttachmentReader(file string, r io.Reader) Mailer {
	c.attachmentReaders = make(map[string]readerInfo)
	c.attachmentReaders[file] = readerInfo{r: r}
	return c
}

// AttachmentInlineReader set email inline attachment
func (c *customerio) AttachmentInlineReader(file string, r io.Reader) Mailer {
	c.attachmentInlineReaders = make(map[string]readerInfo)
	c.attachmentInlineReaders[file] = readerInfo{r: r}
	return c
}

// Send process an email sending
func (c *customerio) Send() error {
	// verify params for sending email
	c.verifyParams()

	//check the total file size and path
	var totalSize int64
	for _, f := range c.attachmentFiles {
		fi, e := os.Stat(f)
		if e != nil {
			return e
		}
		// get the size
		totalSize += fi.Size()
	}

	if totalSize > customerioMaxFileSize {
		return errors.New("gomailer: max attachment size for customerio is 30MB")
	}

	// build attachment
	for _, f := range c.attachmentFiles {
		a := attachment{}
		err := a.ReadFromFile(f)
		if err != nil {
			return err
		}
		a.Disposition = "attachment"
		c.attachments = append(c.attachments, a)
	}

	// build attachment
	for f, r := range c.attachmentReaders {
		a := attachment{}
		err := a.ReadFromReader(f, r)
		if err != nil {
			return err
		}
		a.Disposition = "attachment"
		c.attachments = append(c.attachments, a)
	}

	// build attachment inine
	for _, f := range c.attachmentInlineFiles {
		a := attachment{}
		err := a.ReadFromFile(f)
		if err != nil {
			return err
		}
		a.Disposition = "inline"
		c.attachments = append(c.attachments, a)
	}

	// build attachment inine
	for f, r := range c.attachmentInlineReaders {
		a := attachment{}
		err := a.ReadFromReader(f, r)
		if err != nil {
			return err
		}
		a.Disposition = "inline"
		c.attachments = append(c.attachments, a)
	}
	req := cio.SendEmailRequest{
		From:    c.from.format(),
		To:      c.lists(c.toList),
		Subject: c.subject,

		Identifiers: map[string]string{
			"id": uuid.New().String(),
		},
	}

	if len(c.ccList) > 0 {
		req.BCC = c.lists(c.ccList)
	}

	if len(c.bccList) > 0 {
		req.BCC = c.lists(c.bccList)
	}

	if c.replyTo.Email != "" {
		req.ReplyTo = c.replyTo.format()
	}

	if c.bodyText != "" {
		req.PlaintextBody = c.bodyText
	}

	if c.bodyHTML != "" {
		req.Body = c.bodyHTML
	}

	if len(c.attachments) > 0 {
		files := map[string]string{}
		for _, a := range c.attachments {
			files[a.FileName] = a.Content
		}
		req.Attachments = files
	}

	ctx := context.Background()
	client := cio.NewAPIClient(c.configs.APIKey, cio.WithRegion(cio.RegionUS))
	if _, err := client.SendEmail(ctx, &req); err != nil {
		return err
	}

	return nil
}

func (c customerio) lists(a []address) string {
	if len(a) <= 0 {
		return ""
	}
	list := []string{}
	for _, v := range a {
		list = append(list, v.format())
	}
	return strings.Join(list, ",")
}

// verifyParams verify the required params
func (c customerio) verifyParams() {
	if c.from.Email == "" {
		panic("gomailer: you must provide from")
	}
	if len(c.toList) <= 0 {
		panic("gomailer: you must provide at least one receipent")
	}
	if len(c.toList)+len(c.ccList)+len(c.bccList) > customerioMaxReceipents {
		panic(fmt.Sprintf("mailer: total number of receipents including to/cc/bcc can not be greater than %d for customerio", customerioMaxReceipents))
	}
	if c.bodyText == "" && c.bodyHTML == "" {
		panic("gomailer: you must provide a Text or HTML body")
	}
}
