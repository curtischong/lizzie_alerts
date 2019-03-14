package main

/*
You have to make the bash scripts executables by using this command:
	chmod +x
	reference this page: https://stackoverflow.com/questions/25834277/executing-a-bash-script-from-golang
*/

import (
  "strconv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

func readLastSeen(path string) string {
	lastSeen, err := ioutil.ReadFile(path + "lastSeen.txt")
	if err != nil {
		fmt.Println(err.Error())
	}
	contents := string(lastSeen)
	return contents
}

func handleCalls(path string, slackurl string) {
	fmt.Println("Executing query")
	out, err := exec.Command(path + "getCompetitions.sh").Output()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("processing query")
	var data [][]string
	var kernelRef []string
	rows := strings.Split(string(out), "\n")

  if strings.Contains(rows[0], "502 - Bad Gateway") {
    //The connection is down rn. better to wait another 4 hours than to crash
    return
  }

  //Why are we printing this? bc if the curl command gives us an error we'll know
  fmt.Println("The first res is: ", rows[0])

	for i := 1; i < len(rows); i++ {
    if rows[i] != ""{
      cells := strings.Split(rows[i], ",")
      data = append(data, cells)
      kernelRef = append(kernelRef, cells[0])
    }
	}

	fmt.Println("reading last seenFile")
	// read from file
	lastSeenContents := readLastSeen(path)
	if lastSeenContents == "" {
		fmt.Println("No kernels have been observed yet. Using new kernels")
		// use new kernels
	}
	lastSeenContentsArr := strings.Split(lastSeenContents, ",")
	if len(lastSeenContentsArr) == 0 || kernelRef[0] == lastSeenContentsArr[0] {
		fmt.Println("No new updates.")
		// no new kernels trigger loop here
		return
	}
	numNewKernels := 0
	for i := 0; i < len(kernelRef); i++ {
		if kernelRef[i] != lastSeenContentsArr[0] {
			numNewKernels++
		}else{
      break
    }
	}
	fmt.Printf("found %d new kernels.", numNewKernels)
	// save to file
	fmt.Println("saving new kernels")
	newKernelsSeen := strings.Join(kernelRef, ",")
	err = ioutil.WriteFile("lastSeen.txt", []byte(newKernelsSeen), 0644)
	if err != nil {
		panic(err)
	}
	fmt.Println("saved new kernels")

	// format data to send to slack
	message := "{\"text\": \"```" + strconv.Itoa(numNewKernels) + " New Kernel(s):\n"
	for i := 0; i < numNewKernels; i++ {
		message += ("https://kaggle.com/" + data[i][0] + "\n")
		message += (data[i][1] + "\n")
		message += (data[i][2] + "\n")
		message += (data[i][4])// + "\n\n"
		if i != numNewKernels-1 {
			message += ("\n\n")
		}
	}
	message += ("```\"}")
	sendToSlack(slackurl, message)
}

func sendToSlack(webhookurl string, message string) {
	fmt.Println("Sending new kernel alert to slack")
  fixedStr := strings.NewReader(message)
	req, err := http.NewRequest("POST", webhookurl, fixedStr)
	req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	//fmt.Println("response Status:", resp.Status)
	//fmt.Println("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))
}

type Config struct {
	Webhookurl string `json:"webhookurl"`
}

func LoadConfiguration(file string) Config {
	fmt.Println("Loading config file")
	var config Config
	configFile, err := os.Open(file)
	defer configFile.Close()
	if err != nil {
		fmt.Println(err.Error())
	}
	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)
	return config
}

func main() {

	const path string = "/usr/local/go/src/github.com/curtischong/lizzie_alerts/kaggleKernelWorker/"
	config := LoadConfiguration(path + "config.json")

	const durationBetweenCalls = 2 * time.Hour
	for {
		go handleCalls(path, config.Webhookurl)
		time.Sleep(durationBetweenCalls)
	}
}
