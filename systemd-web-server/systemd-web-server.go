package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"time"
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

func sendAction(host, port, service, action string) ([]byte, error) {

	postCmd := PostCommand{Command: action, Service: service}
	postCmdBytes, err := json.Marshal(postCmd)
	if err != nil {
		return nil, err
	}

	host = fmt.Sprintf("http://%s:%s/", host, port)
	resp, err := http.Post(host, "application/json", bytes.NewBuffer(postCmdBytes))
	if err != nil {
		return nil, err
	} else if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			body = []byte(fmt.Sprintf("Could not read body, err: %v", err))
		}
		return nil, fmt.Errorf("Actions %s for %s service on %s:%s host returned %d; %s", action, service, host, port, resp.StatusCode, body)
	}

	return ioutil.ReadAll(resp.Body)
}

func systemdaction(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/action" {
		w.WriteHeader(http.StatusBadRequest)
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	host := "localhost"
	port := "8081"

	switch r.Method {
	case "POST":
		action := r.PostFormValue("action")
		service := r.PostFormValue("service")
		fmt.Fprintf(w, "%s\n", action)
		fmt.Fprintf(w, "%s\n", service)

		output, err := sendAction(host, port, service, action)
		if err != nil {
			fmt.Fprintf(w, "Error executing %s for %s on %s:%s; %s, %v", action, service, host, port, output, err)
		}

		time.Sleep(500 * time.Millisecond)
		status, err := sendAction(host, port, service, "status")
		if err != nil {
			fmt.Fprintf(w, "Error executing status for %s on %s:%s; %s, %v", service, host, port, status, err)
		}

		fmt.Fprintf(w, "%s\n", status)
	default:
		fmt.Fprintf(w, "Sorry, method %v is not supported, only POST is supported.", r.Method)
	}

}

func systemdlist(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/list" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	host := "localhost"
	port := "8081"

	switch r.Method {
	case "GET":
		resp, err := http.Get(fmt.Sprintf("http://%s:%s", host, port))
		if err != nil {
			log.Printf("Could not retrive services list from %s:%s", host, port)
			fmt.Fprintf(w, "Could not retrive services list from %s:%s", host, port)
			return
		}

		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Could not read response body for services list on %s:%s", host, port)
			fmt.Fprintf(w, "Could not read response body for services list on %s:%s", host, port)
			return
		}

		services := []Service{}

		err = json.Unmarshal(b, &services)
		if err != nil {
			log.Printf("Could not unmarshal response body to services struct on %s:%s", host, port)
			fmt.Fprintf(w, "Could not unmarshal response body to services struct on %s:%s", host, port)
			return
		}

		ss := Services{}
		ss.Services = services
		tmpl := template.Must(template.ParseFiles("layout.html"))
		err = tmpl.Execute(w, ss)
		if err != nil {
			log.Printf("Could not fill in the layout template for services from %s:%s", host, port)
			fmt.Fprintf(w, "Could not fill in the layout template for services from %s:%s", host, port)
			return
		}
	default:
		fmt.Fprintf(w, "Sorry, only GET and POST methods are supported.")
	}

}

func main() {

	http.HandleFunc("/list", systemdlist)
	http.HandleFunc("/action", systemdaction)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
