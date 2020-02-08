package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

type Service struct {
	Service string
	Load    string
	Active  string
	Sub     string
	Desc    string
}

type PostCommand struct {
	Command string
	Service string
}

func hello(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	switch r.Method {
	case "GET":
		services := getSystemServices()

		serviceJSON, err := json.Marshal(services)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Fprintf(w, "%s", serviceJSON)
	case "POST":
		b, err := ioutil.ReadAll(r.Body)
		var postCmd = PostCommand{}
		err = json.Unmarshal(b, &postCmd)
		if err != nil {
			log.Fatal(err)
		}

		out, err := ExecSystemCtl(postCmd)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Fprintf(w, "%s\n", out)

	default:
		fmt.Fprintf(w, "Sorry, only GET and POST methods are supported.")
	}

}

func ExecSystemCtl(postCmd PostCommand) (string, error) {

	command := fmt.Sprintf("systemctl %s %s", postCmd.Command, postCmd.Service)
	out, err := exec.Command("sh", "-c", command).Output()
	if err != nil {
		log.Fatal(err)
	}
	return string(out), err
}

func getSystemServices() []Service {
	out, err := exec.Command("sh", "-c", "systemctl --type=service --all").Output()
	if err != nil {
		log.Fatal(err)
	}

	lines := strings.Split(string(out), "\n")
	//fmt.Printf("%s\n", out)

	var header string
	if len(lines) > 0 {
		header = lines[0]
		lines = lines[1:]
	}

	lastLine := 0
	for i := len(lines); i > 0; i-- {
		if len(lines[i-1]) == 0 {
			lastLine = i - 1
		}
	}

	servicesInd := strings.Index(header, "UNIT")     // UNIT LOAD ACTIVE SUB DESCRIPTION
	unitInd := strings.Index(header, "LOAD")         // UNIT LOAD ACTIVE SUB DESCRIPTION
	activeInd := strings.Index(header, "ACTIVE")     // UNIT LOAD ACTIVE SUB DESCRIPTION
	subInd := strings.Index(header, "SUB")           // UNIT LOAD ACTIVE SUB DESCRIPTION
	descpInd := strings.Index(header, "DESCRIPTION") // UNIT LOAD ACTIVE SUB DESCRIPTION

	var services = []Service{}

	for i, l := range lines {
		if i >= lastLine {
			break
		}

		var s Service
		s.Service = l[servicesInd:unitInd]
		s.Service = strings.TrimSpace(s.Service)

		s.Load = l[unitInd:activeInd]
		s.Load = strings.TrimSpace(s.Load)

		s.Active = l[activeInd:subInd]
		s.Active = strings.TrimSpace(s.Active)

		s.Sub = l[subInd:descpInd]
		s.Sub = strings.TrimSpace(s.Sub)

		s.Desc = l[descpInd:]
		s.Desc = strings.TrimSpace(s.Desc)

		services = append(services, s)

	}

	return services
}

func main() {

	http.HandleFunc("/", hello)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}

	/*
		for _, s := range services {
			//fmt.Printf("SER %s, LOAD %s, ACT %s, SUB %s, DESC %s", s.Service, s.Load, s.Active, s.Sub, s.Desc)
			fmt.Printf("SER %s", s.Service)
			fmt.Println()
			fmt.Printf("LOAD %10s ", s.Load)
			fmt.Printf("ACT %10s ", s.Active)
			fmt.Printf("SUB %10s ", s.Sub)
			fmt.Printf("DESC %10s", s.Desc)
			fmt.Println()
			fmt.Println()
		}*/
}
