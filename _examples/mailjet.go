package main

import (
	"log"

	mailer "github.com/thedevsaddam/gomailer"
)

func main() {
	// new mailer
	c := mailer.Configs{
		PublicKey:  "Your mailjet public key",
		PrivateKey: "Your mailjet private key",
	}
	m, err := mailer.New(mailer.MAILJET, c)
	checkError(err)

	m.From("John Doe", "john@mail.com")
	m.To("Jane Doe", "jane@mail.com")
	m.Cc("Tom", "tom@mail.com")
	m.Cc("Jerry", "jerry@mail.com") // you can add multiple CC, BCC
	m.Bcc("Batman", "batman@mail.com")
	m.ReplyTo("Iron man", "iman@mail.com")

	m.Subject("This is an urgent email")
	// m.BodyText("email with attachment") // if you have plain text body
	m.BodyHTML("<html>Inline image here: <img src='cid:a.png'></html>")
	m.AttachmentFile("a.png")

	checkError(err)
	err = m.Send()
	checkError(err)
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
