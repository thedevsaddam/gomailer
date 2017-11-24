Package gomailer
==================
[![Go Report Card](https://goreportcard.com/badge/github.com/thedevsaddam/gomailer)](https://goreportcard.com/report/github.com/thedevsaddam/gomailer)
[![GoDoc](https://godoc.org/github.com/thedevsaddam/gomailer?status.svg)](https://godoc.org/github.com/thedevsaddam/gomailer)
[![License](https://img.shields.io/dub/l/vibe-d.svg)](https://github.com/thedevsaddam/gomailer/blob/dev/LICENSE.md)

This package provides a simple email interface to integrate third party email services. You can integrate popular email services and switch easily.

### Installation

Install the package using
```go
$ go get github.com/thedevsaddam/gomailer
```

### Usage

To use the package import it in your `*.go` code
```go
import "github.com/thedevsaddam/gomailer"
```
### Integration

```go
package main

import (
	"log"

	mailer "github.com/thedevsaddam/gomailer"
)

func main() {
	c := mailer.Configs{
		Domain: "Your domain here",
		APIKey: "Your key here",
	}

	m, err := mailer.New(mailer.MAILGUN, c)
	checkError(err)

	m.From("John Doe", "john@example.com")
	m.To("Jane Doe", "jane@example.com")
	m.Cc("Tom", "tom@example.com")
	m.Bcc("Jerry", "jerry@example.com")
  	m.ReplyTo("Salman", "joye@example.com")
	m.Subject("mailgun: Urgent email about tom & jerry")
	// m.BodyText("This is a test text email")
	m.BodyHTML("<html>Inline image here: <img src='cid:a.jpg'></html>")
	m.AttachmentFile("a.jpg")

	checkError(err)
	err = m.Send()
	checkError(err)
}

func checkError(err error) {
	if err != nil {
		log.Println(err)
	}
}

```

***You can chain methods easily***

```go
m, err := mailer.New(mailer.MAILGUN, c)
m.To("name", "email").Cc("name", "email").Bcc("name", "email").Bcc("name", "email").Subject("Your subject").BodyText("simple message here").AttachmentFile("some/file.zip").Send()
```

### More [examples](_examples/)

### Roadmap
- [x] Mailgun
- [x] Sendgrid
- [x] Postmark
- [x] Mailjet
- [ ] Elasticmail
- [ ] Jangomail
- [ ] Leadersend
- [ ] Madmimi
- [ ] Mandrill
- [ ] Postageapp
- [ ] Socketlabs
- [ ] Sparkpost

### Note
This package is under development, need to write tests, unimplemented services. Use now at your own risk.

### Contribution
Your suggestions will be more than appreciated.
[Read the contribution guide here](CONTRIBUTING.md)

### See all [contributors](https://github.com/thedevsaddam/gomailer/graphs/contributors)

### Read [API doc](https://godoc.org/github.com/thedevsaddam/gomailer) to know about ***Available options and Methods***

### **License**
The **gomailer** is an open-source software licensed under the [MIT License](LICENSE.md).
