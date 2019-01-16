package main

/*
You have to make the bash scripts executables by using this command:
	chmod +x
	reference this page: https://stackoverflow.com/questions/25834277/executing-a-bash-script-from-golang
*/

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

func main() {
	path := "/Users/curtis/go/src/github.com/curtischong/lizzie_alerts/kaggleKernelWorker/getCompetitions.sh"
	//out, err := exec.Command(path, "#!/bin/sh").Output()
	out, err := exec.Command(path).Output()
	if err != nil {
		log.Fatal(err)
	}
	var data [][]string
	rows := strings.Split(string(out), "\n")

	for i := 0; i < len(rows); i++ {
		cells := strings.Split(rows[i], ",")
		data = append(data, cells)
	}

	for i := 0; i < len(data[0]); i++ {
		fmt.Println(data[0][i])
	}
	fmt.Println(data[1][0])
}
