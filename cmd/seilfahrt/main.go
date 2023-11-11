package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"cgt.name/pkg/go-mwclient"
	"github.com/pelletier/go-toml"
)

type Config struct {
	HedgedocURL    string
	WikiURL        string
	PlenenPageId   string
	ConsumerToken  string
	ConsumerSecret string
	AccessToken    string
	AccessSecret   string
}

var hedgedocPadID string
var configPath string

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

func download(id string, conf Config) (string, error) {
	protourl := fmt.Sprintf(conf.HedgedocURL+"/%v/download", id)
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
	return result, nil
}

func createPage(filepath string, conf Config) error {
	file, err := os.OpenFile(filepath, os.O_RDWR, 0600)
	if err != nil {
		return errors.New("Error opening file: " + err.Error())
	}
	defer file.Close()
	rawBytes, err := io.ReadAll(file)
	if err != nil {
		return errors.New("Error reading file: " + err.Error())
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
				return errors.New("Error creating Pagetitle from Date:" + err.Error())
			}
			foundcount = foundcount + 1
		}
		if foundcount == 2 {
			break
		}
	}
	fmt.Println("Plenumname: ", plenumname)
	fmt.Println("Creating Plenumspage with Name:", plenumname, "and Number:", plenumsnummer)

	// Initialize a *Client with New(), specifying the wiki's API URL
	// and your HTTP User-Agent. Try to use a meaningful User-Agent.
	w, err := mwclient.New(conf.WikiURL, "seilfahrt")
	if err != nil {
		return errors.New("Error creating client:" + err.Error())
	}

	err = w.OAuth(conf.ConsumerToken, conf.ConsumerSecret, conf.AccessToken, conf.AccessSecret)
	if err != nil {
		return errors.New("Error while oauth:" + err.Error())
	}

	pageCreateParameters := map[string]string{
		"title":  plenumname,
		"format": "json",
		"text":   string(rawBytes),
	}

	err = w.Edit(pageCreateParameters)
	if err != nil {
		return errors.New("Error creating page:" + err.Error())
	}

	fmt.Println("Page created")

	fmt.Println("Updating Plenetarium page")
	linkline := fmt.Sprintf("* [[%v]] #%v", plenumname, plenumsnummer)
	pageEditParameters := map[string]string{
		"id":          conf.PlenenPageId,
		"format":      "json",
		"prependtext": linkline,
	}

	err = w.Edit(pageEditParameters)
	if err != nil {
		return errors.New("Error updating page:" + err.Error())
	}

	fmt.Println("Page updated")

	return nil
}

func main() {
	flag.StringVar(&configPath, "c", "./config.toml", "Path to the config file")
	flag.StringVar(&hedgedocPadID, "id", "", "The id of the pad found in the pads URL")
	flag.Parse()
	if hedgedocPadID == "" {
		log.Fatal("No PadID given")
	}
	conf, err := loadConfig(configPath)
	if err != nil {
		log.Fatal("Error loading config: ", err)
	}
	mdFile, err := download(hedgedocPadID, conf)
	if err != nil {
		log.Fatal("Error downloading: ", err)
	}
	wikiFile, err := convert(mdFile)
	if err != nil {
		log.Fatal("Error converting: ", err)
	}
	err = createPage(wikiFile, conf)
	if err != nil {
		log.Fatal("Error creating wikipage: ", err)
	}
}
