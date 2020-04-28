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
)

// https://dev.mailjet.com/email-api/v3/apikey/

const (
	// baseURL describes maingul mail sending api base url
	mailjetBaseURL = "https://api.mailjet.com/v3.1"
	// mailjetMaxFileSize describes the max file size in bytes for sending per email for mailjet
	mailjetMaxFileSize int64 = 15 * 1000000
	// mailjetMaxReceipents describes the max receipents per email
	mailjetMaxReceipents = 50
)

type (
	// mailjet describes a mailjet type
	mailjet struct {
		c                     client
		configs               Configs
		from                  mailjetAddress
		toList                []mailjetAddress
		ccList                []mailjetAddress
		bccList               []mailjetAddress
		replyTo               mailjetAddress
		subject               string
		bodyHTML              string
		bodyText              string
		attachmentFiles       []string
		attachmentInlineFiles []string
	}

	// mailjetAddress represents mailjet address
	mailjetAddress struct {
		Name  string `json:"Name,omitempty"`
		Email string `json:"Email"`
	}
	// attachment describes an email attachment
	mailjetAttachment struct {
		Name        string `json:"Filename"`
		Content     string `json:"Base64Content"`
		ContentType string `json:"ContentType"`
		ContentID   string `json:"ContentID,omitempty"`
	}
)

// messageURL return a message url
func (m *mailjet) messageURL() string {
	url := mailjetBaseURL
	if m.configs.BaseURL != "" {
		url = m.configs.BaseURL
	}
	return fmt.Sprintf("%s/send", url)
}

// From sets an email sender address
func (m *mailjet) From(name, from string) Mailer {
	m.from = mailjetAddress{Name: name, Email: from}
	return m
}

// To sets receipents of an email
func (m *mailjet) To(name, to string) Mailer {
	m.toList = append(m.toList, mailjetAddress{Name: name, Email: to})
	return m
}

// Cc sets CC receipents of an email
func (m *mailjet) Cc(name, to string) Mailer {
	m.ccList = append(m.ccList, mailjetAddress{Name: name, Email: to})
	return m
}

// Bcc sets BCC receipents of an email
func (m *mailjet) Bcc(name, to string) Mailer {
	m.bccList = append(m.bccList, mailjetAddress{Name: name, Email: to})
	return m
}

// ReplyTo sets the reply-to address of an email
func (m *mailjet) ReplyTo(name, email string) Mailer {
	m.replyTo = mailjetAddress{Name: name, Email: email}
	return m
}

// Subject sets subject of an email
func (m *mailjet) Subject(subject string) Mailer {
	m.subject = subject
	return m
}

// BodyHTML sets html body for an email
func (m *mailjet) BodyHTML(body string) Mailer {
	m.bodyHTML = body
	return m
}

// BodyText sets plain text email body for an email
func (m *mailjet) BodyText(body string) Mailer {
	m.bodyText = body
	return m
}

// AttachmentFile set email attachments
func (m *mailjet) AttachmentFile(file string) Mailer {
	m.attachmentFiles = append(m.attachmentFiles, file)
	return m
}

// AttachmentInlineFile set email inline attachment
func (m *mailjet) AttachmentInlineFile(file string) Mailer {
	m.attachmentInlineFiles = append(m.attachmentInlineFiles, file)
	return m
}

// AttachmentReader set email attachments
func (m *mailjet) AttachmentReader(file string, r io.Reader) Mailer {
	log.Println("gomailer: not implemented")
	return m
}

// AttachmentInlineReader set email inline attachment
func (m *mailjet) AttachmentInlineReader(file string, r io.Reader) Mailer {
	log.Println("gomailer: not implemented")
	return m
}

// Send process an email sending
func (m *mailjet) Send() error {
	// verify params for sending email
	m.verifyParams()

	//check the total file size and path
	var totalSize int64
	totalFiles := []string{}
	totalFiles = append(totalFiles, m.attachmentFiles...)
	totalFiles = append(totalFiles, m.attachmentFiles...)
	for _, f := range totalFiles {
		fi, e := os.Stat(f)
		if e != nil {
			return e
		}
		totalSize += fi.Size()
	}

	if totalSize > mailjetMaxFileSize {
		return errors.New("gomailer: max attachment size for mailjet is 15MB")
	}

	// build attachment
	attachments := []mailjetAttachment{}
	for _, f := range m.attachmentFiles {
		a := attachment{}
		err := a.ReadFromFile(f)
		if err != nil {
			return err
		}
		attachments = append(attachments, mailjetAttachment{
			Name:        a.FileName,
			Content:     a.Content,
			ContentType: a.Type,
		})
	}

	inlinedAttachments := []mailjetAttachment{}
	for _, f := range m.attachmentInlineFiles {
		a := attachment{}
		err := a.ReadFromFile(f)
		if err != nil {
			return err
		}
		inlinedAttachments = append(inlinedAttachments, mailjetAttachment{
			Name:        a.FileName,
			Content:     a.Content,
			ContentType: a.Type,
			ContentID:   a.ContentID,
		})
	}

	// build params
	params := mapData{
		"From":    m.from,
		"Subject": m.subject,
	}

	if len(m.toList) > 0 {
		params["To"] = m.toList
	}

	if len(m.ccList) > 0 {
		params["Cc"] = m.ccList
	}

	if len(m.bccList) > 0 {
		params["Bcc"] = m.bccList
	}

	if m.replyTo.Email != "" {
		params["ReplyTo"] = m.replyTo
	}
	if len(m.bodyText) > 0 {
		params["TextPart"] = m.bodyText
	}

	if len(m.bodyHTML) > 0 {
		params["HTMLPart"] = m.bodyHTML
	}

	if len(m.attachmentFiles) > 0 {
		params["Attachments"] = attachments
	}

	if len(m.attachmentInlineFiles) > 0 {
		params["InlinedAttachments"] = inlinedAttachments
	}

	body := struct {
		Messages []mapData `json:"Messages"`
	}{[]mapData{params}}
	return m.processMailjetRequest(body)
}

// verifyParams verify the required params
func (m mailjet) verifyParams() {
	if m.configs.PrivateKey == "" ||
		m.configs.PublicKey == "" {
		panic("gomailer: for mailjetapp you must provide PrivateKey and PublicKey in config")
	}
	if m.from.Email == "" {
		panic("gomailer: you must provide from")
	}
	if len(m.toList) <= 0 {
		panic("gomailer: you must provide at least one receipent")
	}
	if len(m.toList)+len(m.ccList)+len(m.bccList) > mailjetMaxReceipents {
		panic(fmt.Sprintf("mailer: total number of receipents including to/cc/bcc can not be greater than %d for mailjet", mailjetMaxReceipents))
	}
	if m.bodyText == "" && m.bodyHTML == "" {
		panic("gomailer: you must provide a Text or HTML body")
	}
}

// processMailjetRequest perform a post request with content type application/json for mailjet
func (m *mailjet) processMailjetRequest(bodyParams interface{}) error {
	body, err := toJSON(bodyParams)
	if err != nil {
		return err
	}

	req, errReq := http.NewRequest("POST", m.messageURL(), bytes.NewBuffer(body))
	if errReq != nil {
		return errReq
	}

	req.SetBasicAuth(m.configs.PublicKey, m.configs.PrivateKey)
	req.Header.Add("Content-Type", "application/json")

	resp, err := m.c.getDefaultClient().Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return errors.New(string(body))
	}
	return nil
}

// {
//         "Messages":[
//                 {
//                         "From": {
//                                 "Email": "saddam@pathao.com",
//                                 "Name": "Mailjet Pilot"
//                         },
//                         "To": [
//                                 {
//                                         "Email": "saddam@pathao.com",
//                                         "Name": "passenger 1"
//                                 }
//                         ],
//                         "Cc": [
//                                 {
//                                         "Email": "thedevsaddam@gmail.com",
//                                         "Name": "passenger 1"
//                                 }
//                         ],
//                          "Bcc": [
//                                 {
//                                         "Email": "saddam.cse.gu@gmail.com",
//                                         "Name": "passenger 1"
//                                 }
//                         ],
//                         "Attachments": [
//                                 {
//                                         "ContentType": "text/plain",
//                                         "Filename": "test.txt",
//                                         "Base64Content": "VGhpcyBpcyB5b3VyIGF0dGFjaGVkIGZpbGUhISEK"
//                                 }
//                         ],
//                         "InlinedAttachments": [
//                                 {
//                                         "ContentType": "image/gif",
//                                         "Filename": "logo.gif",
//                                         "ContentID": "id1",
//                                         "Base64Content": "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"
//                                 }
//                         ],
//                         "Subject": "Your email flight plan!",
//                         "TextPart": "Dear passenger 1, welcome to Mailjet! May the delivery force be with you!",
//                         "HTMLPart": "<h3>Dear passenger 1, welcome to Mailjet!</h3><br />May the delivery force be with you!"
//                 }
//         ]
// }
