# GO SYTEMD REMOTE CONTROL

This project allows user to control sytemd services remotely. It contains go systemd server and go systemd client.

![Hosts](/screenshots/hosts.png)

![Services](/screenshots/services.png)

![Service Status](/screenshots/status.png)


## GO SYSTEMD CLIENT
_________________________________

Client is machine with REST API that sits on GNU/Linux machine with systemd and serves list of all systemd processes. It can also start/stop/restart/send status of the service on demand.
To be able to exeute start/stop/restart, it should be ran with elevated privilages.

Command line arguments are:

  * listen-ip string - (default "")
    * IP where server will listen for client. Leave empty to listen all IP addresses.
  * listen-port string - (default "8081")
    * Port where server will listen for clients.

Example:
`./systemd-web-client -listen-ip=127.0.0.1 -listen-port=8001`

## GO SYSTEMD SERVER
________________________________

Server is machine which pings for the clients on the given IP range and ask them for the list of the services. It can also ask the client to start/stop/restart/send status of the service.

Command line arguments are:

  * cidr string - (default "192.168.1.1/24")
    * Enter IP range you want to scan - example: `192.168.1.115/32` will try to reach only `192.168.1.115` and `192.168.1.1/24` will try to reach `192.168.1.*` 
  * listen-ip string - (default "")
    * IP where server will listen for client. Leave empty to listen all IP addresses.
  * listen-port string - (default "8080")
    * Port where server will listen for clients.
  * scan-port string -(default "8081")
    * Port on which it will try to reach for the clients.

Example:
`./systemd-web-server -listen-port=8000 -scan-port=8001 -listen-ip=127.0.0.1 -cidr=127.0.0.1/32`
