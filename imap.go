package main

import (
	"fmt"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message"
	"io"
	"io/ioutil"
	"log"
)

func fetchEmails(imapClient *client.Client) error {
	// Select the mailbox you want to read
	mailbox, err := imapClient.Select("INBOX", false)
	if err != nil {
		return err
	}

	// Define the range of emails to fetch
	seqSet := new(imap.SeqSet)
	seqSet.AddRange(1, mailbox.Unseen)

	// Fetch the required message attributes
	messages := make(chan *imap.Message, 10)
	section := &imap.BodySectionName{}
	items := []imap.FetchItem{section.FetchItem(), imap.FetchEnvelope}

	go func() {
		if err := imapClient.Fetch(seqSet, items, messages); err != nil {
			log.Fatal(err)
		}
	}()

	//for msg := range messages {
	//	fmt.Println("Subject:", msg.Envelope.Subject)
	//	body := msg.Body
	//	for b := range body {
	//		text := body[b]
	//		bod, _ := ioutil.ReadAll(text)
	//		tex := string(bod)
	//		fmt.Println(tex[strings.Index(tex, "Content-Type: text/html; charset=\"utf-8\""):strings.Index(tex[strings.Index(tex, "Content-Type: text/html; charset=\"utf-8\""):], "--===============")])
	//	}
	//	break
	//}

	if err != nil {
		log.Println(err)
	}
	seqset := new(imap.SeqSet)
	for msg := range messages {
		seqset.AddNum(msg.Uid)
		for _, r := range msg.Body {
			entity, err := message.Read(r)
			if err != nil {
				log.Fatal(err)
			}
			multiPartReader := entity.MultipartReader()
			if multiPartReader == nil {
				continue
			}
			for e, err := multiPartReader.NextPart(); err != io.EOF; e, err = multiPartReader.NextPart() {

				if err != nil {
					break
				}
				kind, params, cErr := e.Header.ContentType()
				if cErr != nil {
					log.Fatal(cErr)
				}
				if kind == "text/html" {
					t, _ := ioutil.ReadAll(e.Body)
					log.Println(string(t))
				}
				if kind == "image/png" || kind == "image/gif" {
					c, rErr := ioutil.ReadAll(e.Body)
					if rErr != nil {
						log.Fatal(rErr)
					}

					log.Printf("Dump file %s", params["name"])

					if fErr := ioutil.WriteFile("/tmp/"+params["name"], c, 0777); fErr != nil {
						log.Fatal(fErr)
					}
				}

			}
		}
	}

	section = &imap.BodySectionName{}
	items = []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchInternalDate, section.FetchItem()}
	messages = make(chan *imap.Message)
	err = imapClient.Fetch(seqset, items, messages)

	return nil
}

func connectToServer(username, password, server string, port int) (*client.Client, error) {
	c, err := client.DialTLS(fmt.Sprintf("%s:%d", server, port), nil)
	if err != nil {
		return nil, err
	}

	if err := c.Login(username, password); err != nil {
		return nil, err
	}

	return c, nil
}

func main3141() {
	username := "wyncher"
	password := "ahtyodbxjyjcnspe"
	server := "imap.yandex.ru"
	port := 993

	imapClient, err := connectToServer(username, password, server, port)
	_ = fetchEmails(imapClient)
	if err != nil {
		log.Fatal(err)
	}
	defer imapClient.Logout()
	//to := "recipient@example.com"
	//subject := "Test Email"
	//body := "This is a test email sent from a Go-based email client."

	//if err := sendEmail(username, password, server, 587, to, subject, body); err != nil {
	//	log.Fatal(err)
	//}
}
