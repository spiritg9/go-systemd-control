package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"
)

const client_appname = "go-systemd-client"

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

type Hosts struct {
	Hosts []Host
}

type Host struct {
	Hostname string
	AppName  string
	Version  string
	IP       string
	Port     string
}

var hosts = make(map[string]Host)

func sendAction(host, port, service, action string) ([]byte, error) {

	postCmd := PostCommand{Command: action, Service: service}
	postCmdBytes, err := json.Marshal(postCmd)
	if err != nil {
		return nil, err
	}

	host = fmt.Sprintf("http://%s:%s/services", host, port)
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

func home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	switch r.Method {
	case "GET":
		tmpl := template.Must(template.ParseFiles("layoutHosts.html"))

		var h = Hosts{}
		for _, value := range hosts {
			h.Hosts = append(h.Hosts, value)
		}

		err := tmpl.Execute(w, h)
		if err != nil {
			log.Printf("Could not fill in the layout template for hosts; %v", err)
			fmt.Fprintf(w, "Could not fill in the layout template for hosts; %v", err)
			return
		}
	default:
		fmt.Fprintf(w, "Sorry, only GET and POST methods are supported.")
	}

}

func systemdlist(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/list" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	switch r.Method {
	case "GET":
		hosts, ok := r.URL.Query()["host"]
		if !ok || len(hosts[0]) < 1 {
			w.WriteHeader(http.StatusBadRequest)
			log.Println("Url Param 'host' is missing")
			fmt.Fprintf(w, "Url Param 'host' is missing")
			return
		}
		ports, ok := r.URL.Query()["port"]
		if !ok || len(ports[0]) < 1 {
			w.WriteHeader(http.StatusBadRequest)
			log.Println("Url Param 'port' is missing")
			fmt.Fprintf(w, "Url Param 'port' is missing")
			return
		}

		host := hosts[0]
		port := ports[0]

		resp, err := http.Get(fmt.Sprintf("http://%s:%s/services", host, port))
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
			log.Printf("Could not fill in the layout template for services from %s:%s; %v", host, port, err)
			fmt.Fprintf(w, "Could not fill in the layout template for services from %s:%s %v, ", host, port, err)
			return
		}
	default:
		fmt.Fprintf(w, "Sorry, only GET and POST methods are supported.")
	}

}

func searchNewHosts(cidr, port string) {
	var ips = getIPs(cidr)
	fmt.Println(ips)
	for {
		for _, ip := range ips {
			go pingServer(ip, port)
		}

		time.Sleep(60 * time.Second)
	}

}

func pingServer(ip, port string) {
	log.Printf("pinging %s:%s", ip, port)
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Get(fmt.Sprintf("http://%s:%s", ip, port))
	if err != nil {
		log.Printf("Could not get response from %s:%s", ip, port)
		return
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Could not read response body from %s:%s", ip, port)
		return
	}

	var host Host
	err = json.Unmarshal(b, &host)
	host.IP = ip
	host.Port = port
	if err != nil {
		log.Printf("Did not recognize %s:%s as go-systemd-client, server returned %s", ip, port, b)
	} else if host.AppName == client_appname {
		log.Printf("Found %s:%s, hostname: %s, app version %s, app name %s", ip, port, host.Hostname, host.Version, host.AppName)
		key := fmt.Sprintf("%s%s", ip, port)
		hosts[key] = host
	}
}

func getIPs(cidr string) []string {
	var ipList = []string{}
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		log.Fatal(err)
	}
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		ipList = append(ipList, fmt.Sprintf("%v", ip))
	}
	return ipList
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func main() {

	cidr := flag.String("cidr", "192.168.1.1/24", "example: 192.168.1.115/32 or 192.168.1.1/24")
	port := flag.String("scan-port", "8081", "8081")
	listenIP := flag.String("listen-ip", "", "for 0.0.0.0 leave empty")
	listenPort := flag.String("listen-port", "8080", "8080")

	flag.Parse()

	go searchNewHosts(*cidr, *port)

	http.HandleFunc("/", home)
	http.HandleFunc("/list", systemdlist)
	http.HandleFunc("/action", systemdaction)

	listenAddr := fmt.Sprintf("%s:%s", *listenIP, *listenPort)
	if err := http.ListenAndServe(listenAddr, nil); err != nil {
		log.Fatal(err)
	}
}
