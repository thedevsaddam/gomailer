package gomailer

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	// baseURL describes mailgun mail sending api base url
	mailgunBaseURL = "https://api.mailgun.net/v3"
	// maxFileSize describes the max file size in bytes for sending per email for mailgun
	mailgunMaxFileSize int64 = 25 * 1000000
	// mailgunMaxReceipents describes the max receipents per email
	mailgunMaxReceipents = 1000
)

// mailgun describes a mailgun type
type mailgun struct {
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
}

// lists return a formatted email list comma separate string
func (mailgun) lists(a []address) string {
	if len(a) <= 0 {
		return ""
	}
	list := []string{}
	for _, v := range a {
		list = append(list, v.format())
	}
	return strings.Join(list, ",")
}

// messageURL return a message url
func (m *mailgun) messageURL() string {
	url := mailgunBaseURL
	if m.configs.BaseURL != "" {
		url = m.configs.BaseURL
	}
	if m.configs.Domain == "" {
		panic("gomailer: you must provide domain name in Config")
	}
	return fmt.Sprintf("%s/%s/messages", url, m.configs.Domain)
}

// From sets an email sender address
func (m *mailgun) From(name, from string) Mailer {
	m.from = address{Name: name, Email: from}
	return m
}

// To sets receipents of an email
func (m *mailgun) To(name, to string) Mailer {
	m.toList = append(m.toList, address{Name: name, Email: to})
	return m
}

// Cc sets Cc receipents of an email
func (m *mailgun) Cc(name, to string) Mailer {
	m.ccList = append(m.ccList, address{Name: name, Email: to})
	return m
}

// Bcc sets Bcc receipents of an email
func (m *mailgun) Bcc(name, to string) Mailer {
	m.bccList = append(m.bccList, address{Name: name, Email: to})
	return m
}

// ReplyTo sets the reply-to address of an email
func (m *mailgun) ReplyTo(name, email string) Mailer {
	m.replyTo = address{Name: name, Email: email}
	return m
}

// Subject sets subject of an email
func (m *mailgun) Subject(subject string) Mailer {
	m.subject = subject
	return m
}

// BodyHTML sets html body for an email
func (m *mailgun) BodyHTML(body string) Mailer {
	m.bodyHTML = body
	return m
}

// BodyText sets plain text email body for an email
func (m *mailgun) BodyText(body string) Mailer {
	m.bodyText = body
	return m
}

// AttachmentFile set email attachments
func (m *mailgun) AttachmentFile(file string) Mailer {
	m.attachmentFiles = append(m.attachmentFiles, file)
	return m
}

// AttachmentInlineFile set email inline attachment
func (m *mailgun) AttachmentInlineFile(file string) Mailer {
	m.attachmentInlineFiles = append(m.attachmentInlineFiles, file)
	return m
}

// Send process an email sending
func (m *mailgun) Send() error {
	// verify params for sending email
	m.verifyParams()

	//check the total file size
	var totalSize int64
	for _, f := range m.attachmentFiles {
		fi, e := os.Stat(f)
		if e != nil {
			return e
		}
		// get the size
		totalSize += fi.Size()
	}

	if totalSize > mailgunMaxFileSize {
		return errors.New("gomailer: max attachment size for mailgun is 25MB")
	}

	// build params
	params := map[string]string{
		"from":    m.from.format(),
		"to":      m.lists(m.toList),
		"subject": m.subject,
	}
	if len(m.ccList) > 0 {
		params["cc"] = m.lists(m.ccList)
	}
	if len(m.bccList) > 0 {
		params["bcc"] = m.lists(m.bccList)
	}
	if m.replyTo.Email != "" {
		params["h:Reply-To"] = m.replyTo.format()
	}
	if m.bodyText != "" {
		params["text"] = m.bodyText
	}
	if m.bodyHTML != "" {
		params["html"] = m.bodyHTML
	}

	// build attachments for both inline and general attachments
	attachments := map[string][]string{}
	attachments["attachment"] = m.attachmentFiles
	attachments["inline"] = m.attachmentInlineFiles

	return m.processMailgunRequest(params, attachments)
}

// verifyParams verify the required params
func (m mailgun) verifyParams() {
	if m.from.Email == "" {
		panic("gomailer: you must provide from")
	}
	if len(m.toList) <= 0 {
		panic("gomailer: you must provide at least one receipent")
	}
	if len(m.toList)+len(m.ccList)+len(m.bccList) > mailgunMaxReceipents {
		panic(fmt.Sprintf("mailer: total number of receipents including to/cc/bcc can not be greater than %d for mailgun", mailgunMaxReceipents))
	}
	if m.bodyText == "" && m.bodyHTML == "" {
		panic("gomailer: you must provide a Text or HTML body")
	}
}

// processMailgunRequest build a post request for mailgun
func (m *mailgun) processMailgunRequest(params map[string]string, files map[string][]string) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// add files if exist
	for fileFieldName, fPaths := range files {
		for index, fPath := range fPaths {
			file, errF := os.Open(fPath)
			if errF != nil {
				return errF
			}
			paramName := fmt.Sprintf("%s[%d]", fileFieldName, index)
			part, err := writer.CreateFormFile(paramName, filepath.Base(fPath))
			if err != nil {
				return err
			}
			_, err = io.Copy(part, file)
			if err != nil {
				return err
			}
			if err := file.Close(); err != nil {
				return err
			}
		}
	}

	// add extra params
	for key, val := range params {
		_ = writer.WriteField(key, val)
	}

	err := writer.Close()
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", m.messageURL(), body)

	req.Header.Set("Content-Type", writer.FormDataContentType())
	// do basic auth for mailgun
	req.SetBasicAuth("api", m.configs.APIKey)

	// process the post request
	resp, err := m.c.getDefaultClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bodyByte, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New(string(bodyByte))
	}
	return err
}
