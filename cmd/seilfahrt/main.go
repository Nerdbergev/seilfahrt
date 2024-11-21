package main

import (
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"cgt.name/pkg/go-mwclient"
	"github.com/Nerdbergev/seilfahrt/internal/htmltemplates"
	"github.com/pelletier/go-toml"
	"github.com/google/uuid"
	gomail "gopkg.in/mail.v2"
)

type Config struct {
	WikiURL        string
	PlenenPageId   string
	ConsumerToken  string
	ConsumerSecret string
	AccessToken    string
	AccessSecret   string
	SendMail       bool
	MailAdress     string
	MailPassword   string
	MailServer     string
	MailPort       int
	MailRecipient  string
	MailSubject    string
	MailTemplate   string
}

type ResponseData struct {
	Title   string
	Message string
}

type MailContent struct {
	Link string
}

var hedgedocPad string
var configPath string
var port int
var web bool

var sumbitTmpl = template.Must(template.New("submit").Parse(htmltemplates.SubmitTemplate))
var responseTmpl = template.Must(template.New("response").Parse(htmltemplates.ResponseTemplate))

var conf Config

func loadConfig(filepath string) (Config, error) {
	var result Config
	f, err := os.OpenFile(filepath, os.O_RDONLY, 0600)
	if err != nil {
		return result, errors.New("Error opening config file:" + err.Error())
	}
	defer f.Close()
	decoder := toml.NewDecoder(f)
	err = decoder.Decode(&result)
	if err != nil {
		return result, errors.New("Error decoding config file:" + err.Error())
	}
	return result, nil
}

func download(urlstring string) (string, error) {
	_, err := url.ParseRequestURI(urlstring)
	if err != nil {
		return "", errors.New("Error parsing url: " + err.Error())
	}
	protourl := urlstring
	if protourl[len(protourl)-1:] == "#" {
		protourl = protourl[:len(protourl)-1]
	}
	protourl = protourl + "/download"
	fmt.Println("Downloading Hedgedoc with url:", protourl)
	client := &http.Client{}
	req, err := http.NewRequest("GET", protourl, nil)
	if err != nil {
		return "", errors.New("Error creating request: " + err.Error())
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.New("Error executing request: " + err.Error())
	}
	fmt.Println("Creating temporary md file for protocoll")
	f, err := os.CreateTemp("", "protocol-*.md")
	if err != nil {
		return "", errors.New("Error creating file: " + err.Error())
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return "", errors.New("Error copying into file: " + err.Error())
	}
	return f.Name(), nil
}

func convert(filepath string) (string, error) {
	fmt.Println("Converting file to mediawiki format")
	outputpath := filepath + ".wiki"
	cmd := exec.Command("pandoc", "-f", "markdown", "-t", "mediawiki", "-o", filepath+".wiki", filepath)
	err := cmd.Run()
	if err != nil {
		return "", errors.New("Error executing command: " + err.Error())
	}
	return outputpath, nil
}

func createPageTitlefromDate(date string) (string, error) {
	result := strings.ReplaceAll(date, "| ", "")
	result = strings.TrimSpace(result)
	t, err := time.Parse("02.01.2006", result)
	if err != nil {
		return "", errors.New("Error parsing date:" + err.Error())
	}
	result = t.Format("2006-01-02")
	result = "Plenum:" + result
	return result, nil
}

func createPage(filepath string, conf Config) (string, error) {
	file, err := os.OpenFile(filepath, os.O_RDWR, 0600)
	if err != nil {
		return "", errors.New("Error opening file: " + err.Error())
	}
	defer file.Close()
	rawBytes, err := io.ReadAll(file)
	if err != nil {
		return "", errors.New("Error reading file: " + err.Error())
	}
	var plenumname string
	var plenumsnummer string
	var foundcount = 0
	lines := strings.Split(string(rawBytes), "\n")
	for i, line := range lines {
		if strings.Contains(line, "! Plenumsnummer") {
			plenumsnummer = strings.ReplaceAll(lines[i+1], "! ", " ")
			plenumsnummer = strings.TrimSpace(plenumsnummer)
			foundcount = foundcount + 1
		}
		if strings.Contains(line, "| Datum") {
			plenumname, err = createPageTitlefromDate(lines[i+1])
			if err != nil {
				return "", errors.New("Error creating Pagetitle from Date:" + err.Error())
			}
			foundcount = foundcount + 1
		}
		if foundcount == 2 {
			break
		}
	}
	if plenumname == "" || plenumsnummer == "" {
		return "", errors.New("error finding plenumname or plenumsnummer")
	}
	fmt.Println("Plenumname: ", plenumname)
	fmt.Println("Creating Plenumspage with Name:", plenumname, "and Number:", plenumsnummer)

	// Initialize a *Client with New(), specifying the wiki's API URL
	// and your HTTP User-Agent. Try to use a meaningful User-Agent.
	w, err := mwclient.New(conf.WikiURL, "seilfahrt")
	if err != nil {
		return "", errors.New("Error creating client:" + err.Error())
	}

	err = w.OAuth(conf.ConsumerToken, conf.ConsumerSecret, conf.AccessToken, conf.AccessSecret)
	if err != nil {
		return "", errors.New("Error while oauth:" + err.Error())
	}

	pageCreateParameters := map[string]string{
		"title":  plenumname,
		"format": "json",
		"text":   string(rawBytes),
	}

	err = w.Edit(pageCreateParameters)
	if err != nil {
		return "", errors.New("Error creating page:" + err.Error())
	}

	fmt.Println("Page created")
	pagename := "https://wiki.nerdberg.de/" + plenumname
	fmt.Println(pagename)

	fmt.Println("Updating Plenetarium page")
	linkline := fmt.Sprintf("* [[%v]] #%v\n", plenumname, plenumsnummer)
	pageEditParameters := map[string]string{
		"pageid":      conf.PlenenPageId,
		"format":      "json",
		"prependtext": linkline,
	}

	err = w.Edit(pageEditParameters)
	if err != nil {
		return "", errors.New("Error updating page:" + err.Error())
	}

	fmt.Println("Page updated")

	return pagename, nil
}

func SendMail(pageName string, conf Config) error {
	mailcontent := MailContent{
		Link: pageName,
	}

	tmpl, err := template.ParseFiles(conf.MailTemplate)
	if err != nil {
		return errors.New("Error parsing template:" + err.Error())
	}
	var body strings.Builder
	err = tmpl.Execute(&body, mailcontent)
	if err != nil {
		return errors.New("Error executing template:" + err.Error())
	}

	m := gomail.NewMessage()

	// Set E-Mail sender
	m.SetHeader("From", conf.MailAdress)

	// Set E-Mail receivers
	m.SetHeader("To", conf.MailRecipient)

	// Set E-Mail subject
	m.SetHeader("Subject", conf.MailSubject)

	// Set E-Mail body. You can set plain text or html with text/html
	m.SetBody("text/plain", body.String())

	// Set Message Id
	m.SetHeader("Message-ID", uuid.New().String())

	// Settings for SMTP server
	d := gomail.NewDialer(conf.MailServer, 587, conf.MailAdress, conf.MailPassword)

	// Now send E-Mail
	if err := d.DialAndSend(m); err != nil {
		return errors.New("Error sending mail:" + err.Error())
	}

	return nil
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	sumbitTmpl.Execute(w, nil)
}

func returnError(w http.ResponseWriter, message string) {
	log.Println(message)
	data := ResponseData{
		Title:   "Error",
		Message: message,
	}
	responseTmpl.Execute(w, data)
}

func submitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		url := r.FormValue("url")
		var data ResponseData

		mdFile, err := download(url)
		if err != nil {
			returnError(w, "Error downloading: "+err.Error())
			return
		}
		wikiFile, err := convert(mdFile)
		if err != nil {
			returnError(w, "Error converting: "+err.Error())
			return
		}
		pagename, err := createPage(wikiFile, conf)
		if err != nil {
			returnError(w, "Error creating wikipage: "+err.Error())
			return
		}

		if conf.SendMail {
			err := SendMail(pagename, conf)
			if err != nil {
				returnError(w, "Error sending mail: "+err.Error())
				return
			}
		}

		data = ResponseData{
			Title:   "Success",
			Message: "Page created",
		}

		responseTmpl.Execute(w, data)
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

func startWebserver() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/submit", submitHandler)
	fmt.Println("Starting webserver on port:", port)
	log.Println(http.ListenAndServe(fmt.Sprintf(":%v", port), nil))
}

func main() {
	flag.StringVar(&configPath, "c", "./config.toml", "Path to the config file")
	flag.StringVar(&hedgedocPad, "pad", "", "The URL to the hedgedoc pad.")
	flag.IntVar(&port, "port", 8080, "Port for the webserver")
	flag.BoolVar(&web, "web", false, "Start the webserver")
	flag.Parse()
	if !web {
		if hedgedocPad == "" {
			log.Fatal("No Pad given")
		}
	}
	var err error
	conf, err = loadConfig(configPath)
	if err != nil {
		log.Fatal("Error loading config: ", err)
	}
	if web {
		startWebserver()
	} else {
		mdFile, err := download(hedgedocPad)
		if err != nil {
			log.Fatal("Error downloading: ", err)
		}
		wikiFile, err := convert(mdFile)
		if err != nil {
			log.Fatal("Error converting: ", err)
		}
		_, err = createPage(wikiFile, conf)
		if err != nil {
			log.Fatal("Error creating wikipage: ", err)
		}
	}
}
