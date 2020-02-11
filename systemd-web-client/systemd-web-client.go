package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

const appname = "go-systemd-client"

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

type Host struct {
	Hostname string
	AppName  string
	Version  string
}

func host(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	switch r.Method {
	case "GET":
		hostname, err := os.Hostname()
		if err != nil {
			log.Printf("Could not retrive hostname; %v", err)
		}

		h := Host{
			Hostname: hostname,
			AppName:  appname,
			Version:  "alpha",
		}

		hostJSON, err := json.Marshal(h)
		if err != nil {
			// if we can't marshal it, we can't send it and server won't find it so no use in running
			log.Fatal("Could not marshal hostname; s", err)
		}

		fmt.Fprintf(w, "%s\n", hostJSON)
	default:
		fmt.Fprintf(w, "Sorry, only GET method is supported.")
		return
	}

}

func systemdserve(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/services" {
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
			fmt.Fprintf(w, "Could not unmarshal POST request JSON; %v; %s", err, b)
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
		temp := []rune(l)[servicesInd:unitInd]
		s.Service = strings.TrimSpace(string(temp))

		temp = []rune(l)[unitInd:activeInd]
		s.Load = strings.TrimSpace(string(temp))

		temp = []rune(l)[activeInd:subInd]
		s.Active = strings.TrimSpace(string(temp))

		temp = []rune(l)[subInd:descpInd]
		s.Sub = strings.TrimSpace(string(temp))

		temp = []rune(l)[descpInd:]
		s.Desc = strings.TrimSpace(string(temp))

		services = append(services, s)
	}

	return services, nil
}

func main() {
	listenIP := flag.String("listen-ip", "", "for 0.0.0.0 leave empty")
	listenPort := flag.String("listen-port", "8081", "8081")

	flag.Parse()

	http.HandleFunc("/", host)
	http.HandleFunc("/services", systemdserve)

	listenAddr := fmt.Sprintf("%s:%s", *listenIP, *listenPort)
	if err := http.ListenAndServe(listenAddr, nil); err != nil {
		log.Fatal(err)
	}
}
