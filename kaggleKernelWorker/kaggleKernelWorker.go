package main

/*
You have to make the bash scripts executables by using this command:
	chmod +x
	reference this page: https://stackoverflow.com/questions/25834277/executing-a-bash-script-from-golang
*/

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func handleCalls(path, slackurl string) {
	fmt.Println("Executing query")
	out, err := exec.Command(path + "getCompetitions.sh").Output()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("processing query")
	var data [][][]byte
	var kernelRef [][]byte
	rows := bytes.Split(out, []byte("\n"))

	if bytes.Contains(rows[0], []byte("502 - Bad Gateway")) {
		// The connection is down rn. better to wait another 4 hours than to crash
		return
	}

	// Why are we printing this? bc if the curl command gives us an error we'll know
	fmt.Println("The first res is: ", string(rows[0]))

	for i := 1; i < len(rows); i++ {
		if len(rows[i]) != 0 {
			cells := bytes.Split(rows[i], []byte(","))
			data = append(data, cells)
			kernelRef = append(kernelRef, cells[0])
		}
	}

	fmt.Println("reading last seenFile")
	lastSeenContents, err := ioutil.ReadFile(path + "lastSeen.txt")
	if err != nil {
		log.Fatal(err)
	}
	if len(lastSeenContents) == 0 {
		fmt.Println("No kernels have been observed yet. Using new kernels")
		// Use new kernels
	}
	lastSeenContentsArr := bytes.Split(lastSeenContents, []byte(","))
	if len(lastSeenContentsArr) == 0 || bytes.Compare(kernelRef[0], lastSeenContentsArr[0]) == 0 {
		fmt.Println("No new updates.")
		// No new kernels trigger loop here
		return
	}
	numNewKernels := 0
	for i := 0; i < len(kernelRef); i++ {
		if bytes.Compare(kernelRef[i], lastSeenContentsArr[0]) == 0 {
			break
		}
		numNewKernels++
	}
	fmt.Printf("found %d new kernels.", numNewKernels)
	// save to file
	fmt.Println("saving new kernels")
	newKernelsSeen := bytes.Join(kernelRef, []byte(","))
	err = ioutil.WriteFile("lastSeen.txt", []byte(newKernelsSeen), 0644)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("saved new kernels")

	var messageBuilder strings.Builder
	// format data to send to slack
	messageBuilder.WriteString("{\"text\": \"```" + strconv.Itoa(numNewKernels) + " New Kernel(s):\n")
	for i := 0; i < numNewKernels; i++ {
		messageBuilder.WriteString("https://kaggle.com/")
		messageBuilder.WriteString(string(data[i][0]))
		messageBuilder.WriteString("\n")
		messageBuilder.WriteString(string(data[i][1]))
		messageBuilder.WriteString("\n")
		messageBuilder.WriteString(string(data[i][2]))
		messageBuilder.WriteString("\n")
		messageBuilder.WriteString(string(data[i][4]))
		if i != numNewKernels-1 {
			messageBuilder.WriteString("\n\n")
		}
	}
	messageBuilder.WriteString("```\"}")
	sendToSlack(slackurl, messageBuilder.String())
}

func sendToSlack(webhookurl, message string) {
	fmt.Println("Sending new kernel alert to slack")
	fixedStr := strings.NewReader(message)
	req, err := http.NewRequest("POST", webhookurl, fixedStr)
	req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))
	if err != nil {
		log.Fatal(err)
	}
}

type Config struct {
	Webhookurl string `json:"webhookurl"`
}

func LoadConfiguration(file string) Config {
	fmt.Println("Loading config file")
	var config Config
	configFile, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	defer configFile.Close()
	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)
	return config
}

func main() {
	const path string = "/Users/curtis/go/src/github.com/curtischong/lizzie_alerts/kaggleKernelWorker/"
	config := LoadConfiguration(path + "config.json")

	const durationBetweenCalls = 2 * time.Hour
	for {
		go handleCalls(path, config.Webhookurl)
		time.Sleep(durationBetweenCalls)
	}
}
