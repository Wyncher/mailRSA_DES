package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/emersion/go-message"
	"github.com/gin-gonic/gin"
	"github.com/jordan-wright/email"
	"github.com/knadh/go-pop3"
	cp "github.com/otiai10/copy"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net/smtp"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func typeofObject(variable interface{}) string {
	switch variable.(type) {
	case []byte:
		return "[]byte"
	case string:
		return "string"
	default:
		return "unknown"
	}
}

func readAccount(mail string) (password string, server string) {
	file, err := os.Open("accounts.txt")
	ErrorHandler(err)
	scanner := bufio.NewScanner(file)
	var user []string
	for scanner.Scan() {
		user = append(user, "")
		if strings.Contains(scanner.Text(), mail) {
			password = strings.Split(scanner.Text(), " ")[1]
			server = strings.Split(scanner.Text(), " ")[2]
			return password, server
		}

	}
	if user[0] == "" && len(user) == 1 {
		log.Println("НЕТУ ТАКОГО ЮЗЕРА. ПОЛЬЗОВАТЕЛЬ ХИТРОВАН")
	}
	return
}
func sendEmail(c *gin.Context, to, subject, body string, attachWay string, attachName string) error {
	//auth check

	username, _ := c.Cookie("account")
	password, server := readAccount(username)
	server = "smtp" + server
	var encrypt_struct encrypt_struct
	encrypt_struct = encryptdes(body, encrypt_struct)
	encrypt_struct = loadSenderPrivateRSAkey(encrypt_struct, username)
	encrypt_struct = loadReceiverPublicRSAkey(encrypt_struct, to)
	encrypt_struct = EncryptOAEP(encrypt_struct)

	if _, err := os.Stat("C:/Users/Wyncher/Desktop/учёба/БПД/course_work_bpd/" + username + "/"); os.IsNotExist(err) {
		err := os.Mkdir("C:/Users/Wyncher/Desktop/учёба/БПД/course_work_bpd/"+username+"/", 0777)
		if err != nil {
			log.Println("Error make dir")
		}
	}
	files, err := ioutil.ReadDir("C:/Users/Wyncher/Desktop/учёба/БПД/course_work_bpd/" + username + "/")
	ErrorHandler(err)
	id := 1
	for _, f := range files {
		if f.Name()[:6] == "out___" {
			id++
		}
	}
	idString := strconv.Itoa(id)
	//attach
	var signature_attach []byte
	if attachWay != "" && attachName != "" {
		cp.Copy(attachWay+attachName, username+"/out___"+idString+attachName)
		f, err := os.Open(username + "/out___" + idString + attachName)
		if err != nil {
			log.Println("Error attach")

		}
		attach, err := io.ReadAll(f)
		signature_attach = SignPSS(encrypt_struct, attach)
	} else {
		file, err := c.FormFile("attach")
		if err != nil {
			log.Println("Error attach")

		}
		c.SaveUploadedFile(file, username+"/out___"+idString+"/"+file.Filename)
		attachName = "/" + file.Filename
	}
	var signature_body = SignPSS(encrypt_struct, []byte(body))
	saveFile(encrypt_struct.des_encrypt_key[:], username+"/out___"+idString, "/key.key")
	saveFile(encrypt_struct.iv_des, username+"/out___"+idString, "/iv_des.iv_des")
	saveFile(encrypt_struct.body_encrypt, username+"/out___"+idString, "/body_encrypt.body_encrypt")
	saveFile(signature_body, username+"/out___"+idString, "/body_encrypt.signature")
	saveFile(signature_attach, username+"/out___"+idString, "/attach.signature")
	saveFile([]byte(body), username+"/out___"+idString, "/body.html")

	t := time.Now()
	if err != nil {
		log.Println(err)
	}

	info := append([]byte(t.Format("2006-1-2 15:4:5")))
	info = append(info, []byte("\n")...)
	info = append(info, []byte(username)...)
	info = append(info, []byte("\n")...)
	info = append(info, []byte(subject)...)
	if fErr := ioutil.WriteFile(username+"/out___"+idString+"/info.txt", info, 0777); fErr != nil {
		log.Fatal(fErr)
	}
	e := email.NewEmail()
	e.From = username
	e.To = []string{to}
	e.Subject = subject
	e.Text = []byte("text")
	e.HTML = []byte("text")
	f, err := os.Open(username + "/out___" + idString + attachName)
	ErrorHandler(err)
	_, err = e.Attach(f, attachName, "")
	f, err = os.Open(username + "/out___" + idString + "/key.key")
	ErrorHandler(err)
	_, err = e.Attach(f, "key.key", "")
	ErrorHandler(err)
	f, err = os.Open(username + "/out___" + idString + "/iv_des.iv_des")
	ErrorHandler(err)
	_, err = e.Attach(f, "iv_des.iv_des", "")
	f, err = os.Open(username + "/out___" + idString + "/body_encrypt.body_encrypt")
	ErrorHandler(err)
	_, err = e.Attach(f, "body_encrypt.body_encrypt", "")
	f, err = os.Open(username + "/out___" + idString + "/body_encrypt.signature")
	ErrorHandler(err)
	_, err = e.Attach(f, "body_encrypt.signature", "")
	f, err = os.Open(username + "/out___" + idString + "/attach.signature")
	ErrorHandler(err)
	_, err = e.Attach(f, "attach.signature", "")
	if err != nil {
		return err
	}
	err = e.Send(server+":587", smtp.PlainAuth("", username, password, server))
	if err != nil {
		return err
	} else {
		log.Println("Send OK")
	}

	return nil
}
func getFrom(header message.Header) (from string) {
	reg := regexp.MustCompile("(<).*(>)")
	fromArr := reg.FindAllString(header.Get("From"), -1)
	if len(fromArr) == 0 {
		return ""
	}
	from = fromArr[0]
	return from[1 : len(from)-1]
}
func pop(user string, c *pop3.Conn) (lettersArray []map[string]interface{}) {

	defer c.Quit()

	//make list file if not exist
	if _, err := os.Stat(user + "/list.txt"); errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(user+"/", 0777)
		//generateRSAkeys(user)
		_, err := os.Create(user + "/list.txt")
		if err != nil {
			log.Println(err)
		}
	}
	//Open list file with ID's
	b, err := ioutil.ReadFile(user + "/list.txt")
	if err != nil {
		log.Println(err)
	}

	// Print the total number of messages and their size.
	f, err := os.OpenFile(user+"/list.txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()

	// Mark list as readed
	msgs, _ := c.List(0)
	for _, m := range msgs {

		var id = strconv.Itoa(m.ID)
		if strings.Contains(string(b), id) == false {

			if _, err = f.WriteString(id + "\n"); err != nil {
				fmt.Println("error write")
				log.Println(err)
			}

			m, _ := c.Retr(m.ID)

			date := m.Header.Get("Date")
			subjectRaw := m.Header.Get("Subject")

			var attachName string
			//subject, _ := base64.StdEncoding.DecodeString(strings.Split(subjectRaw, " ")[0])
			dec := new(mime.WordDecoder)
			subject, err := dec.DecodeHeader(subjectRaw)
			if err != nil {
				log.Println(err)
			}

			filename := user + "/income"
			filename += id
			err = os.MkdirAll(filename, 0777)
			_, err = os.Create(filename + "/info.txt")
			if err != nil {
				log.Println(err)
			}
			info := append([]byte(date))
			info = append(info, []byte("\n")...)
			info = append(info, []byte(getFrom(m.Header))...)
			info = append(info, []byte("\n")...)
			info = append(info, string(subject)...)
			if fErr := ioutil.WriteFile(filename+"/info.txt", info, 0777); fErr != nil {
				fmt.Println("error write info")
				log.Fatal(fErr)
			}
			reader := m.MultipartReader()
			for e, err := reader.NextPart(); err != io.EOF; e, err = reader.NextPart() {
				kind, _, cErr := e.Header.ContentType()
				if cErr != nil {
					fmt.Println("error read header")
					log.Fatal(cErr)
				}

				if kind == "multipart/alternative" || kind == "text/html" {
					t, _ := ioutil.ReadAll(e.Body)
					buffer := string(t)
					body := buffer[strings.Index(buffer, "Content-Type: text/html")+42 : len(buffer)-66]
					//body = body[strings.Index(body, "Content-Transfer-Encoding: base64\r\n\r\n"):]
					//decode_body, _ := base64.StdEncoding.DecodeString(body[37:])
					decode_body := body
					filenameBody := filename + "/body.html"
					_, err := os.Create(filenameBody)
					if err != nil {
						log.Println(err)
					}
					if fErr := ioutil.WriteFile(filenameBody, []byte(decode_body), 0777); fErr != nil {
						if fErr := ioutil.WriteFile(filenameBody, []byte(buffer), 0777); fErr != nil {
							log.Fatal(fErr)
						}

						//append to letters array

					}
				}
				if kind == "text/plain" {
					t, _ := ioutil.ReadAll(e.Body)
					body := string(t)
					filenameBody := filename + "/body.html"
					_, err := os.Create(filenameBody)
					if err != nil {
						log.Println(err)
					}
					if fErr := ioutil.WriteFile(filename, []byte(body), 0777); fErr != nil {
						log.Fatal(fErr)
					}

					//append to letters array
					//letter := map[string]interface{}{"id": id, "from": getFrom(m.Header), "to": user, "subject": string(subject), "date": date, "body": template.HTML(filenameBody)}
					//lettersArray = append(lettersArray, letter)
				}

				if strings.Contains(kind, "application") {
					c, rErr := ioutil.ReadAll(e.Body)
					if rErr != nil {
						log.Fatal(rErr)
					}

					_, nameRaw, _ := e.Header.ContentDisposition()
					//if nameRaw["filename"] == "" {
					//	filenameAttache := filename + "/" + "attach" + string(id)
					//} else {
					filenameAttach := filename + "/" + strings.ReplaceAll(nameRaw["filename"], " ", "")
					//log.Printf("Dump file  %s", params["name"])
					_, err := os.Create(filenameAttach)
					if err != nil {
						log.Println(err)
					}
					if fErr := ioutil.WriteFile(filenameAttach, c, 0777); fErr != nil {
						log.Fatal(fErr)
					}
					if nameRaw["filename"] == "iv_des.iv_des" || nameRaw["filename"] == "key.key" || nameRaw["filename"] == "body_encrypt.body_encrypt" || nameRaw["filename"] == "info.txt" || nameRaw["filename"] == "body.html" || strings.Contains(nameRaw["filename"], "signature") {

					} else {
						attachName = strings.ReplaceAll(nameRaw["filename"], " ", "")
					}
				}
				if kind == "image/jpeg" || kind == "image/png" {
					c, rErr := ioutil.ReadAll(e.Body)
					if rErr != nil {
						log.Fatal(rErr)
					}

					_, nameRaw, _ := e.Header.ContentDisposition()
					//if nameRaw["filename"] == "" {
					//	filenameAttache := filename + "/" + "attach" + string(id)
					//} else {
					filenameAttach := filename + "/" + strings.ReplaceAll(nameRaw["filename"], " ", "")
					//}
					//log.Printf("Dump file  %s", params["name"])
					_, err := os.Create(filenameAttach)
					if err != nil {
						log.Println(err)
					}
					if fErr := ioutil.WriteFile(filenameAttach, c, 0777); fErr != nil {
						log.Fatal(fErr)
					}
				}
			}

			var decrypt_struct decrypt_struct
			decrypt_struct.des_encrypt_key = loadFile(user + "/income" + id + "/key.key")
			decrypt_struct.iv_des = loadFile(user + "/income" + id + "/iv_des.iv_des")
			decrypt_struct.cryptoText_des = loadFile(user + "/income" + id + "/body_encrypt.body_encrypt")

			decrypt_struct = loadRecieverPrivateRSAkey(decrypt_struct, user)
			decrypt_struct = DecryptOAEP(decrypt_struct)
			decrypt_struct = decryptdes(decrypt_struct)
			//fmt.Println("Расшифрованная фраза : " + string(decrypt_struct.body_decrypt))
			decrypt_struct = loadSenderPublicRSAkey(decrypt_struct, getFrom(m.Header))
			fmt.Println(VerifyPSS(decrypt_struct, loadFile(user+"/income"+id+"/body_encrypt.signature")))
			fmt.Println(attachName)
			fmt.Println(VerifyPSSAttach(decrypt_struct, loadFile(user+"/income"+id+attachName), loadFile(user+"/income"+id+"/attach.signature")))
			if fErr := ioutil.WriteFile(user+"/income"+id+"/body.html", decrypt_struct.body_decrypt, 0777); fErr != nil {
				log.Fatal(fErr)
			}
			letter := map[string]interface{}{"id": id, "from": getFrom(m.Header), "to": user, "subject": subject, "date": date, "body": string(decrypt_struct.body_decrypt)}
			lettersArray = append(lettersArray, letter)
		}
	}
	fmt.Println(lettersArray)
	return lettersArray
}
func auth(username string, password string, server string) (c *pop3.Conn) {
	f, err := os.Open("C:/Users/Wyncher/Desktop/учёба/БПД/course_work_bpd/accounts.txt")
	defer f.Close()
	data, err := io.ReadAll(f)
	ErrorHandler(err)
	if strings.Contains(string(data), username) == false {
		f.Close()
		f, err := os.OpenFile("C:/Users/Wyncher/Desktop/учёба/БПД/course_work_bpd/accounts.txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			log.Println(err)
		}

		if _, err = f.WriteString(username + " " + password + " " + server); err != nil {
			log.Println(err)
		}
	}
	// Initialize the client.
	p := pop3.New(pop3.Opt{
		Host:       server,
		Port:       995,
		TLSEnabled: true,
	})
	//New connection
	c, err = p.NewConn()
	ErrorHandler(err)

	// Authenticate.
	if err := c.Auth(username, password); err != nil {
		log.Fatal(err)
	}
	return c

}

func main1() {
	var z = auth("milkachikalatka@rambler.ru", "ahtyodbxjyjcnspe324GF", "pop.rambler.ru")
	pop("milkachikalatka@rambler.ru", z)
}
