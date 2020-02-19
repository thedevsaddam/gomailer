package gomailer

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

// https://sendgrid.com/solutions/email-api/

const (
	// baseURL describes maingul mail sending api base url
	sendgridBaseURL = "https://api.sendgrid.com/v3"
	// sendgridMaxFileSize describes the max file size in bytes for sending per email for sendgrid
	sendgridMaxFileSize int64 = 30 * 1000000
	// sendgridMaxReceipents describes the max receipents per email
	sendgridMaxReceipents = 1000
)

type (
	readerInfo struct {
		size int64
		r    io.Reader
	}
	// sendgrid describes a sendgrid type
	sendgrid struct {
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

	sendgridContent struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	}
)

// messageURL return a message url
func (s *sendgrid) messageURL() string {
	url := sendgridBaseURL
	if s.configs.BaseURL != "" {
		url = s.configs.BaseURL
	}
	return fmt.Sprintf("%s/mail/send", url)
}

// From sets an email sender address
func (s *sendgrid) From(name, from string) Mailer {
	s.from = address{Name: name, Email: from}
	return s
}

// To sets receipents of an email
func (s *sendgrid) To(name, to string) Mailer {
	s.toList = append(s.toList, address{Name: name, Email: to})
	return s
}

// Cc sets Cc receipents of an email
func (s *sendgrid) Cc(name, to string) Mailer {
	s.ccList = append(s.ccList, address{Name: name, Email: to})
	return s
}

// Bcc sets Bcc receipents of an email
func (s *sendgrid) Bcc(name, to string) Mailer {
	s.bccList = append(s.bccList, address{Name: name, Email: to})
	return s
}

// ReplyTo sets the reply-to address of an email
func (s *sendgrid) ReplyTo(name, email string) Mailer {
	s.replyTo = address{Name: name, Email: email}
	return s
}

// Subject sets subject of an email
func (s *sendgrid) Subject(subject string) Mailer {
	s.subject = subject
	return s
}

// BodyHTML sets html body for an email
func (s *sendgrid) BodyHTML(body string) Mailer {
	s.bodyHTML = body
	return s
}

// BodyText sets plain text email body for an email
func (s *sendgrid) BodyText(body string) Mailer {
	s.bodyText = body
	return s
}

// AttachmentFile set email attachments
func (s *sendgrid) AttachmentFile(file string) Mailer {
	s.attachmentFiles = append(s.attachmentFiles, file)
	return s
}

// AttachmentInlineFile set email inline attachment
func (s *sendgrid) AttachmentInlineFile(file string) Mailer {
	s.attachmentInlineFiles = append(s.attachmentInlineFiles, file)
	return s
}

// AttachmentReader set email attachments
func (s *sendgrid) AttachmentReader(file string, r io.Reader) Mailer {
	s.attachmentReaders = make(map[string]readerInfo)
	s.attachmentReaders[file] = readerInfo{r: r}
	return s
}

// AttachmentInlineReader set email inline attachment
func (s *sendgrid) AttachmentInlineReader(file string, r io.Reader) Mailer {
	s.attachmentInlineReaders = make(map[string]readerInfo)
	s.attachmentInlineReaders[file] = readerInfo{r: r}
	return s
}

// Send process an email sending
func (s *sendgrid) Send() error {
	// verify params for sending email
	s.verifyParams()

	//check the total file size and path
	var totalSize int64
	for _, f := range s.attachmentFiles {
		fi, e := os.Stat(f)
		if e != nil {
			return e
		}
		// get the size
		totalSize += fi.Size()
	}

	if totalSize > sendgridMaxFileSize {
		return errors.New("gomailer: max attachment size for sendgrid is 30MB")
	}

	// build attachment
	for _, f := range s.attachmentFiles {
		a := attachment{}
		err := a.ReadFromFile(f)
		if err != nil {
			return err
		}
		a.Disposition = "attachment"
		s.attachments = append(s.attachments, a)
	}

	// build attachment
	for f, r := range s.attachmentReaders {
		a := attachment{}
		err := a.ReadFromReader(f, r)
		if err != nil {
			return err
		}
		a.Disposition = "attachment"
		s.attachments = append(s.attachments, a)
	}

	// build attachment inine
	for _, f := range s.attachmentInlineFiles {
		a := attachment{}
		err := a.ReadFromFile(f)
		if err != nil {
			return err
		}
		a.Disposition = "inline"
		s.attachments = append(s.attachments, a)
	}

	// build attachment inine
	for f, r := range s.attachmentInlineReaders {
		a := attachment{}
		err := a.ReadFromReader(f, r)
		if err != nil {
			return err
		}
		a.Disposition = "inline"
		s.attachments = append(s.attachments, a)
	}

	// build params
	params := mapData{
		"from": s.from,
	}

	var to []address

	if len(s.toList) > 0 {
		to = s.toList
	}

	var cc, bcc []address

	if len(s.ccList) > 0 {
		cc = s.ccList
	}
	if len(s.bccList) > 0 {
		bcc = s.bccList
	}

	personalizations := []mapData{
		{
			"to":      to,
			"cc":      cc,
			"bcc":     bcc,
			"subject": s.subject,
		},
	}

	params["personalizations"] = personalizations

	if s.replyTo.Email != "" {
		params["reply_to"] = s.replyTo
	}
	var sendgridContents []sendgridContent
	if len(s.bodyText) > 0 {
		sendgridContents = append(sendgridContents, sendgridContent{
			Type:  "text/plain",
			Value: s.bodyText,
		})
	}

	if len(s.bodyHTML) > 0 {
		sendgridContents = append(sendgridContents, sendgridContent{
			Type:  "text/html",
			Value: s.bodyHTML,
		})
	}

	params["content"] = sendgridContents

	// add attachment if exist
	if len(s.attachments) > 0 {
		params["attachments"] = s.attachments
	}

	return s.processSendgridRequest(params)
}

// verifyParams verify the required params
func (s sendgrid) verifyParams() {
	if s.from.Email == "" {
		panic("gomailer: you must provide from")
	}
	if len(s.toList) <= 0 {
		panic("gomailer: you must provide at least one receipent")
	}
	if len(s.toList)+len(s.ccList)+len(s.bccList) > sendgridMaxReceipents {
		panic(fmt.Sprintf("mailer: total number of receipents including to/cc/bcc can not be greater than %d for sendgrid", sendgridMaxReceipents))
	}
	if s.bodyText == "" && s.bodyHTML == "" {
		panic("gomailer: you must provide a Text or HTML body")
	}
}

// processSendgridRequest perform a post request with content type application/json for sendgrid
func (s *sendgrid) processSendgridRequest(bodyParams map[string]interface{}) error {
	body, err := toJSON(bodyParams)
	if err != nil {
		return err
	}
	req, errReq := http.NewRequest("POST", s.messageURL(), bytes.NewBuffer(body))

	if errReq != nil {
		return errReq
	}

	if s.configs.APIKey != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", s.configs.APIKey))
	}

	req.Header.Add("Content-Type", "application/json")

	resp, err := s.c.getDefaultClient().Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusAccepted {
		body, _ := ioutil.ReadAll(resp.Body)
		return errors.New(string(body))
	}

	return err
}
