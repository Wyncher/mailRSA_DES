package main

import (
	"bufio"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/knadh/go-pop3"
	cp "github.com/otiai10/copy"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// var username = "milkachikalatka@rambler.ru"
// var password = "ahtyodbxjyjcnspe324GF"
// var server = "pop.rambler.ru"
func ErrorHandler(err error) bool {
	if err != nil {
		return true
	} else {
		return false
	}
}
func errorPage(c *gin.Context) {
	c.HTML(http.StatusFound, "error.html", gin.H{})
}
func login(c *gin.Context) {
	quit := c.Query("quit")
	if quit != "" {
		username, _ := c.Cookie("account")
		if username == "" {
			fmt.Println("Не авторизирован")
			c.Redirect(http.StatusFound, "/")
			return
		}
		err := os.RemoveAll(username)
		ErrorHandler(err)
		err = os.RemoveAll("keys/" + username)
		ErrorHandler(err)
		file, err := os.Open("C:/Users/Wyncher/Desktop/учёба/БПД/course_work_bpd/accounts.txt")

		scanner := bufio.NewScanner(file)
		var newFile string
		for scanner.Scan() {
			fmt.Println(scanner.Text())
			if strings.Contains(scanner.Text(), username) != true {
				newFile += scanner.Text() + "\n"
			}
		}
		file.Close()
		err = os.RemoveAll("C:/Users/Wyncher/Desktop/учёба/БПД/course_work_bpd/accounts.txt")
		ErrorHandler(err)
		file, err = os.Create("C:/Users/Wyncher/Desktop/учёба/БПД/course_work_bpd/accounts.txt")
		ErrorHandler(err)
		_, err = file.Write([]byte(newFile[:len(newFile)-2]))
		ErrorHandler(err)

		file.Close()
		c.SetCookie("account", "", 360, "/", "127.0.0.1", true, true)
	}
	email := c.Query("email")
	password := c.Query("password")

	if email != "" && password != "" {

		f, err := os.Open("C:/Users/Wyncher/Desktop/учёба/БПД/course_work_bpd/accounts.txt")

		data, err := io.ReadAll(f)
		if err != nil {
			log.Fatal(err)
		}
		f.Close()
		if strings.Contains(string(data), email) == false {
			f, err := os.OpenFile("C:/Users/Wyncher/Desktop/учёба/БПД/course_work_bpd/accounts.txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
			defer f.Close()
			if err != nil {
				panic(err)
			}
			var server string
			emailSplit := strings.Split(email, "@")
			if len(emailSplit) != 2 {
				c.HTML(http.StatusForbidden, "login.html", gin.H{})
			}

			switch emailSplit[1] {
			case "yandex.com":
				server = ".yandex.ru"
			case "yandex.ru":
				server = ".yandex.ru"
			case "rambler.ru":
				server = ".rambler.ru"
			case "mail.ru":
				server = ".mail.ru"
			case "inbox.ru":
				server = ".mail.ru"
			case "mail.com":
				server = ".mail.ru"
			default:
				c.HTML(http.StatusForbidden, "login.html", gin.H{})
			}
			if _, err = f.WriteString("\n" + email + " " + password + " " + server); err != nil {
				panic(err)
			}
		}
		generateRSAkeys(email)
		c.SetCookie("account", email, 360, "/", "127.0.0.1", true, true)
		c.Redirect(http.StatusFound, "/inbox")
	}

	c.HTML(http.StatusOK, "login.html", gin.H{})

}
func inbox(c *gin.Context) {
	//auth check
	username, _ := c.Cookie("account")
	if username == "" {
		fmt.Println("Не авторизирован")
		c.Redirect(http.StatusFound, "/")
		return
	}
	//keys change?
	keys := c.Query("keys")
	if keys != "" {
		generateRSAkeys(username)
	}
	//sync?
	sync := c.Query("sync")
	var listLetters []map[string]interface{}
	var authObject *pop3.Conn
	dirs, err := ioutil.ReadDir(username)
	if err != nil || sync != "" {
		password, server := readAccount(username)
		authObject = auth(username, password, "pop3"+server)
		listLetters = pop(username, authObject)
	}

	directionType := c.Query("dir")
	if directionType == "" {
		directionType = "income"
	}
	var cacheListLetters []map[string]interface{}
	var from string
	var date string
	var subject string
	//var bodyByte []byte
	//var body string

	for _, f := range dirs {
		if f.IsDir() && f.Name()[:6] == directionType {
			files, err := ioutil.ReadDir(username + "/" + f.Name())
			if ErrorHandler(err) {
				c.Redirect(http.StatusFound, "/error")
			}

			dir := f.Name()
			for _, f := range files {

				if f.Name() == "info.txt" {
					file, err := os.Open(username + "/" + dir + "/info.txt")
					if err != nil {
						log.Fatal(err)
					}
					id, _ := strconv.Atoi(dir[6:])
					scanner := bufio.NewScanner(file)
					scanner.Scan()
					date = scanner.Text()
					scanner.Scan()
					from = scanner.Text()
					scanner.Scan()
					subject = scanner.Text()
					if err := scanner.Err(); err != nil {
						log.Fatal(err)
					}
					file.Close()
					//f, err := os.Open(username + "/" + dir + "/body.html")
					//if err != nil {
					//	log.Fatal(err)
					//}
					//bodyByte, _ = io.ReadAll(f)
					//f.Close()
					//body = string(bodyByte)
					letter := map[string]interface{}{"from": from, "id": id, "to": username, "subject": subject, "date": date, "type": directionType}
					cacheListLetters = append(cacheListLetters, letter)
				}
			}
		}
	}
	if len(listLetters) == 0 {
		listLetters = append(cacheListLetters)
	} else {
		listLetters = append(listLetters, cacheListLetters...)
	}

	c.HTML(http.StatusOK, "lists.html", gin.H{"t": listLetters})
}
func newMessage(c *gin.Context) {
	username, _ := c.Cookie("account")
	if username == "" {
		fmt.Println("Не авторизирован")
		c.Redirect(http.StatusFound, "/")
		return
	}

	c.HTML(http.StatusOK, "new-message.html", gin.H{})
}
func newMessagePOST(c *gin.Context) {
	email := c.PostForm("email")
	subject := c.PostForm("subject")
	body := c.PostForm("textarea")
	if email == "" || subject == "" || body == "" {
		c.Redirect(http.StatusFound, "/error")
		return
	}
	if c.PostForm("draft") == "on" {
		username, _ := c.Cookie("account")
		files, err := ioutil.ReadDir("C:/Users/Wyncher/Desktop/учёба/БПД/course_work_bpd/" + username + "/")
		if err != nil {
			log.Fatal(err)
		}
		id := 1
		for _, f := range files {
			if f.Name()[:6] == "draft_" {
				id++
			}
		}
		idString := strconv.Itoa(id)
		file, err := c.FormFile("attach")
		if err != nil {
			log.Println("Error attach")

		}
		c.SaveUploadedFile(file, username+"/draft_"+idString+"/"+file.Filename)
		t := time.Now()
		info := append([]byte(t.Format("2006-1-2 15:4:5")))
		info = append(info, []byte("\n")...)
		info = append(info, []byte(username)...)
		info = append(info, []byte("\n")...)
		info = append(info, []byte(subject)...)
		if fErr := ioutil.WriteFile(username+"/draft_"+idString+"/info.txt", info, 0777); fErr != nil {
			log.Fatal(fErr)
		}
		if fErr := ioutil.WriteFile(username+"/draft_"+idString+"/body.html", []byte(body), 0777); fErr != nil {
			log.Fatal(fErr)
		}
		c.Redirect(http.StatusFound, "/inbox?dir=draft_")
		return
	} else {
		if err := sendEmail(c, email, subject, body, "", ""); err != nil {
			log.Fatal(err)
		}
		c.Redirect(http.StatusFound, "/inbox?dir=out___")
		return
	}

}
func accounts(c *gin.Context) {

	account := c.Query("account")
	var users []string
	if account == "" {
		file, err := os.Open("accounts.txt")
		if err != nil {
			log.Fatal(err)
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			users = append(users, strings.Split(scanner.Text(), " ")[0])
		}
	} else {
		file, err := os.Open("accounts.txt")
		if err != nil {
			log.Fatal(err)
		}
		scanner := bufio.NewScanner(file)
		var user []string
		for scanner.Scan() {
			user = append(user, "")
			if strings.Contains(scanner.Text(), account) {
				user = append(strings.Split(scanner.Text(), " "))
			}
			users = append(users, strings.Split(scanner.Text(), " ")[0])
		}
		if user[0] == "" && len(user) == 1 {
			log.Println("НЕТУ ТАКОГО ЮЗЕРА. ПОЛЬЗОВАТЕЛЬ ХИТРОВАН")
		} else {
			c.SetCookie("account", user[0], 360, "/", "127.0.0.1", true, true)
			c.Redirect(http.StatusFound, "/inbox")

		}
	}
	c.HTML(http.StatusOK, "accounts.html", gin.H{"u": users})
}
func listView(c *gin.Context) {
	username, _ := c.Cookie("account")
	if username == "" {
		fmt.Println("Не авторизирован")
		c.Redirect(http.StatusFound, "/")
		return
	}
	draftId := c.Query("draft")
	deleteId := c.Query("delete")
	if deleteId != "" {
		fmt.Println(username + "/income" + deleteId)
		if draftId != "" {
			cp.Copy(username+"/draft_"+deleteId, username+"/delete"+deleteId)
			os.RemoveAll(username + "/draft_" + deleteId)
		} else {
			cp.Copy(username+"/income"+deleteId, username+"/delete"+deleteId)
			os.RemoveAll(username + "/income" + deleteId)
		}

		c.Redirect(http.StatusFound, "/inbox?dir=delete")
		return
	}
	id := c.Query("id")
	directionType := c.Query("dir")
	download := c.Query("download")

	if draftId != "" && download == "" {

		f, err := os.Open("C:/Users/Wyncher/Desktop/учёба/БПД/course_work_bpd/" + username + "/draft_" + draftId + "/body.html")
		body, err := io.ReadAll(f)
		if err != nil {
			log.Fatal(err)
		}
		f.Close()
		file, err := os.Open("C:/Users/Wyncher/Desktop/учёба/БПД/course_work_bpd/" + username + "/draft_" + draftId + "/info.html")

		scanner := bufio.NewScanner(file)
		scanner.Scan()
		_ = scanner.Text()
		scanner.Scan()
		email := scanner.Text()
		scanner.Scan()
		subject := scanner.Text()
		f.Close()
		dirs, err := ioutil.ReadDir(username + "/out___" + draftId)
		var attachWay string
		var attachName string
		for _, f := range dirs {
			if strings.Contains(f.Name(), "body.html") || strings.Contains(f.Name(), "info.txt") {
				continue
			} else {
				attachWay = username + "/draft_" + draftId
				attachName = "/" + f.Name()
			}
		}
		//saveFile()
		if err := sendEmail(c, email, subject, string(body), attachWay, attachName); err != nil {
			log.Fatal(err)
		}
		c.Redirect(http.StatusFound, "/inbox?dir=out___")
		return
	}
	if download != "" {
		dirs, _ := ioutil.ReadDir(username)
		if directionType != "" {
			for _, f := range dirs {
				if draftId != "" {
					id = draftId
				}
				if strings.Contains(f.Name(), id) && f.IsDir() && id != "" && strings.Contains(f.Name(), directionType) {
					files, err := ioutil.ReadDir(username + "/" + f.Name())
					if err != nil {
						log.Fatal(err)
					}
					for _, f := range files {

						if strings.Contains(f.Name(), "body") || strings.Contains(f.Name(), "info") || strings.Contains(f.Name(), "attach") || strings.Contains(f.Name(), "encrypt") || strings.Contains(f.Name(), "key") || strings.Contains(f.Name(), "iv_des") {
							continue
						} else {
							fmt.Println(f.Name())
							c.File(username + "/" + directionType + id + "/" + f.Name())
							return

						}
					}

				}
				//if strings.Contains(f.Name(), draftId) && f.IsDir() && draftId != "" {
				//	files, err := ioutil.ReadDir(username + "/" + f.Name())
				//	if err != nil {
				//		log.Fatal(err)
				//	}
				//	for _, f := range files {
				//		if strings.Contains(f.Name(), "body") || strings.Contains(f.Name(), "info") || strings.Contains(f.Name(), "attach") || strings.Contains(f.Name(), "encrypt") || strings.Contains(f.Name(), "key") || strings.Contains(f.Name(), "iv_des") {
				//			continue
				//		} else {
				//			c.File(username + "/draft_" + draftId + "/" + f.Name())
				//			return
				//		}
				//	}

				//}
			}
		}
	}

	if id != "" {
		dirs, err := ioutil.ReadDir(username)
		if err != nil {
			log.Fatal(err)
		}

		var from string
		var date string
		var subject string
		var bodyByte []byte
		var body string
		var isDelete = true
		var isDraft = false
		for _, f := range dirs {
			if f.Name() == directionType+id {
				files, err := ioutil.ReadDir(username + "/" + f.Name())
				if err != nil {
					log.Fatal(err)
				}
				dir := f.Name()

				if strings.Contains(f.Name(), "delete") {
					isDelete = false
				}
				if strings.Contains(f.Name(), "draft") {
					isDraft = true

				}
				for _, f := range files {

					if f.Name() == "info.txt" {
						file, err := os.Open(username + "/" + dir + "/info.txt")
						if err != nil {
							log.Fatal(err)
						}

						scanner := bufio.NewScanner(file)
						scanner.Scan()
						date = scanner.Text()
						scanner.Scan()
						from = scanner.Text()
						scanner.Scan()
						subject = scanner.Text()
						if err := scanner.Err(); err != nil {
							log.Fatal(err)
						}
						file.Close()
						f, err := os.Open(username + "/" + dir + "/body.html")
						if err != nil {
							log.Fatal(err)
						}
						bodyByte, _ = io.ReadAll(f)
						f.Close()
						body = string(bodyByte)

					}
				}
			}
		}
		var isOut = false
		if directionType == "out___" {
			isOut = true
		}

		if len(body) == 0 {
			c.HTML(http.StatusOK, "list-view.html", map[string]interface{}{"from": from, "id": id, "body": template.HTML(body), "date": date, "subject": subject, "del": isDelete, "draft": isDraft, "out": isOut, "dir": directionType, "signature": "Проверка подписи не пройдена!"})

		} else {

			c.HTML(http.StatusOK, "list-view.html", map[string]interface{}{"from": from, "id": id, "body": template.HTML(body), "date": date, "subject": subject, "del": isDelete, "draft": isDraft, "out": isOut, "dir": directionType, "signature": "Проверка подписи пройдена!"})

		}
	}
}

func router(r gin.IRouter) {
	r.GET("/", login)
	r.GET("/inbox", inbox)
	r.GET("/new", newMessage)
	r.POST("/new", newMessagePOST)
	r.GET("/accounts", accounts)
	r.GET("/view", listView)
	r.GET("/error", errorPage)
}
func staticLoader(r gin.IRouter) {
	r.Static("/static", "./WEB/static")
	r.Static("/styles", "./WEB/styles")
	r.Static("/js", "./WEB/js")
}

func main() {

	r := gin.Default()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.LoadHTMLFiles("WEB/login.html", "WEB/lists.html", "WEB/accounts.html", "WEB/list-view.html", "WEB/new-message.html", "WEB/error.html")
	staticLoader(r)
	router(r)

	log.Println("Server started")
	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
