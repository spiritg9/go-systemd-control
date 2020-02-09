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

type Services struct {
	Services []Service
}

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

func systemdserve(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	switch r.Method {
	case "GET":
		services, err := getSystemServices()
		if err != nil {
			log.Printf("Could not read system services; %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Could not read system services; %v", err)
			return
		}

		serviceJSON, err := json.Marshal(services)
		if err != nil {
			log.Printf("Could not marshal systm services; %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Could not marshal systm services; %v", err)
			return
		}

		fmt.Fprintf(w, "%s", serviceJSON)
	case "POST":
		b, err := ioutil.ReadAll(r.Body)
		var postCmd = PostCommand{}
		err = json.Unmarshal(b, &postCmd)
		if err != nil {
			log.Printf("Could not unmarshal POST request JSON; %v", err)
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Could not unmarshal POST request JSON; %v", err)
			return
		}

		out, err := ExecSystemCtl(postCmd)
		if err != nil {
			log.Printf("Could not execute system command; %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Could not execute system command; %v", err)
			return
		}

		fmt.Fprintf(w, "%s\n", out)

	default:
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Sorry, only GET and POST methods are supported.")
		return
	}

}

func ExecSystemCtl(postCmd PostCommand) ([]byte, error) {

	command := fmt.Sprintf("systemctl %s %s", postCmd.Command, postCmd.Service)
	output, err := exec.Command("sh", "-c", command).Output()
	if eerror, ok := err.(*exec.ExitError); ok {
		if postCmd.Command == "status" {
			/*
				As we don't want to dead services return error, we check if error is due to status code being 1, 2, 3 or 4

				0 program is running or service is OK
				1 program is dead and /var/run pid file exists
				2 program is dead and /var/lock lock file exists
				3 program is not running
				4 program or service status is unknown
				5-99  reserved for future LSB use
				100-149   reserved for distribution use
				150-199   reserved for application use
				200-254   reserved
			*/
			exitCode := eerror.ExitCode()
			if exitCode >= 0 && exitCode <= 4 {
				return output, nil
			}
		}
	}
	return output, err

}

func getSystemServices() ([]Service, error) {
	out, err := exec.Command("sh", "-c", "systemctl --type=service --all").Output()
	if err != nil {
		return []Service{}, err
	}

	// each service should be in new line
	lines := strings.Split(string(out), "\n")

	// first line is header
	var header string
	if len(lines) > 0 {
		header = lines[0]
		lines = lines[1:]
	} else {
		return nil, fmt.Errorf("error listing system services - output len is 0")
	}

	// find where list of services end (first empty line)
	lastLine := 0
	for i := len(lines); i > 0; i-- {
		if len(lines[i-1]) == 0 {
			lastLine = i - 1
		}
	}

	// fix: indexes are not correct when LOAD is 'not-found'
	// also it would be good idea to check if any of indexes end up with -1 -> index not found
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

	return services, nil
}

func main() {

	http.HandleFunc("/", systemdserve)

	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatal(err)
	}
}