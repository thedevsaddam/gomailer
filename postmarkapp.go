package gomailer

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

// https://postmarkapp.com/developer

const (
	// baseURL describes maingul mail sending api base url
	postmarkBaseURL = "https://api.postmarkapp.com"
	// postmarkMaxFileSize describes the max file size in bytes for sending per email for postmark
	postmarkMaxFileSize int64 = 5 * 1000000
	// postmarkMaxReceipents describes the max receipents per email
	postmarkMaxReceipents = 50
)

type (
	// postmark describes a postmark type
	postmark struct {
		c                     client
		configs               Configs
		from                  address
		toList                []address
		ccList                []address
		bccList               []address
		replyTo               address
		subject               string
		bodyHTML              string
		bodyText              string
		attachmentFiles       []string
		attachmentInlineFiles []string
		attachments           []attachment
	}

	// attachment describes an email attachment
	postmarkAttachment struct {
		Name        string `json:"Name"`
		Content     string `json:"Content"`
		ContentType string `json:"ContentType"`
		ContentID   string `json:"ContentID"`
	}
)

// messageURL return a message url
func (p *postmark) messageURL() string {
	url := postmarkBaseURL
	if p.configs.BaseURL != "" {
		url = p.configs.BaseURL
	}
	return fmt.Sprintf("%s/email", url)
}

// From sets an email sender address
func (p *postmark) From(name, from string) Mailer {
	p.from = address{Name: name, Email: from}
	return p
}

// To sets receipents of an email
func (p *postmark) To(name, to string) Mailer {
	p.toList = append(p.toList, address{Name: name, Email: to})
	return p
}

// Cc sets Cc receipents of an email
func (p *postmark) Cc(name, to string) Mailer {
	p.ccList = append(p.ccList, address{Name: name, Email: to})
	return p
}

// Bcc sets Bcc receipents of an email
func (p *postmark) Bcc(name, to string) Mailer {
	p.bccList = append(p.bccList, address{Name: name, Email: to})
	return p
}

// ReplyTo sets the reply-to address of an email
func (p *postmark) ReplyTo(name, email string) Mailer {
	p.replyTo = address{Name: name, Email: email}
	return p
}

// Subject sets subject of an email
func (p *postmark) Subject(subject string) Mailer {
	p.subject = subject
	return p
}

// BodyHTML sets html body for an email
func (p *postmark) BodyHTML(body string) Mailer {
	p.bodyHTML = body
	return p
}

// BodyText sets plain text email body for an email
func (p *postmark) BodyText(body string) Mailer {
	p.bodyText = body
	return p
}

// AttachmentFile set email attachments
func (p *postmark) AttachmentFile(file string) Mailer {
	p.attachmentFiles = append(p.attachmentFiles, file)
	return p
}

// AttachmentInlineFile set email inline attachment
func (p *postmark) AttachmentInlineFile(file string) Mailer {
	p.attachmentInlineFiles = append(p.attachmentInlineFiles, file)
	return p
}

// AttachmentReader set email attachments
func (p *postmark) AttachmentReader(file string, r io.Reader) Mailer {
	log.Println("gomailer: not implemented")
	return p
}

// AttachmentInlineReader set email inline attachment
func (p *postmark) AttachmentInlineReader(file string, r io.Reader) Mailer {
	log.Println("gomailer: not implemented")
	return p
}

// Send process an email sending
func (p *postmark) Send() error {
	// verify params for sending email
	p.verifyParams()

	//check the total file size and path
	var totalSize int64
	for _, f := range p.attachmentFiles {
		fi, e := os.Stat(f)
		if e != nil {
			return e
		}
		// get the size
		totalSize += fi.Size()
	}

	if totalSize > postmarkMaxFileSize {
		return errors.New("gomailer: max attachment size for postmark is 5MB")
	}

	// build attachment
	for _, f := range p.attachmentFiles {
		a := attachment{}
		err := a.ReadFromFile(f)
		if err != nil {
			return err
		}
		a.Disposition = "attachment"
		p.attachments = append(p.attachments, a)
	}

	// build attachment inine
	for _, f := range p.attachmentInlineFiles {
		a := attachment{}
		err := a.ReadFromFile(f)
		if err != nil {
			return err
		}
		a.Disposition = "inline"
		p.attachments = append(p.attachments, a)
	}

	// build params
	params := mapData{
		"From":    p.from.format(),
		"Subject": p.subject,
	}

	if len(p.toList) > 0 {
		var tList []string
		for _, a := range p.toList {
			tList = append(tList, a.format())
		}
		params["To"] = strings.Join(tList, ",")
	}

	if len(p.ccList) > 0 {
		var cList []string
		for _, a := range p.ccList {
			cList = append(cList, a.format())
		}
		params["Cc"] = strings.Join(cList, ",")
	}

	if len(p.bccList) > 0 {
		var bList []string
		for _, a := range p.bccList {
			bList = append(bList, a.format())
		}
		params["Bcc"] = strings.Join(bList, ",")
	}

	if p.replyTo.Email != "" {
		params["ReplyTo"] = p.replyTo
	}
	if len(p.bodyText) > 0 {
		params["TextBody"] = p.bodyText
	}

	if len(p.bodyHTML) > 0 {
		params["HtmlBody"] = p.bodyHTML
	}

	// add attachment if exist
	if len(p.attachments) > 0 {
		var pAttachments []postmarkAttachment
		for _, a := range p.attachments {
			pAttachments = append(pAttachments, postmarkAttachment{
				Name:        a.FileName,
				Content:     a.Content,
				ContentType: a.Type,
				ContentID:   a.ContentID,
			})
		}
		params["Attachments"] = pAttachments
	}

	return p.processPostmarkRequest(params)
}

// verifyParams verify the required params
func (p postmark) verifyParams() {
	if p.configs.AccountToken == "" &&
		p.configs.ServerToken == "" {
		panic("gomailer: for postmarkapp you must provide AccountToken or ServerToken in config")
	}
	if p.from.Email == "" {
		panic("gomailer: you must provide from")
	}
	if len(p.toList) <= 0 {
		panic("gomailer: you must provide at least one receipent")
	}
	if len(p.toList)+len(p.ccList)+len(p.bccList) > postmarkMaxReceipents {
		panic(fmt.Sprintf("mailer: total number of receipents including to/cc/bcc can not be greater than %d for postmark", postmarkMaxReceipents))
	}
	if p.bodyText == "" && p.bodyHTML == "" {
		panic("gomailer: you must provide a Text or HTML body")
	}
}

// processPostmarkRequest perform a post request with content type application/json for postmark
func (p *postmark) processPostmarkRequest(bodyParams map[string]interface{}) error {
	body, err := toJSON(bodyParams)
	if err != nil {
		return err
	}
	req, errReq := http.NewRequest("POST", p.messageURL(), bytes.NewBuffer(body))

	if errReq != nil {
		return errReq
	}

	if p.configs.AccountToken != "" {
		req.Header.Add("X-Postmark-Account-Token", p.configs.AccountToken)
	}

	if p.configs.ServerToken != "" {
		req.Header.Add("X-Postmark-Server-Token", p.configs.ServerToken)
	}

	req.Header.Add("Content-Type", "application/json")

	resp, err := p.c.getDefaultClient().Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return errors.New(string(body))
	}
	return nil
}
