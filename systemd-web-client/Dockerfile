FROM centos 
COPY . /app
COPY systemd-web-client.service /etc/systemd/system/
RUN systemctl enable systemd-web-client
CMD "/sbin/init"
