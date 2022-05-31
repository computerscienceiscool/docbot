FROM ubuntu:22.04

RUN apt-get update 
RUN apt-get install -y apt-utils vim less
RUN apt-get install -y supervisor 

# RUN apt-get install -y curl wget
# RUN apt-get install -y net-tools traceroute mtr-tiny
# RUN apt-get install -y lsof telnet netcat 
# RUN apt-get install -y openssh-client autossh

COPY docbot /root/docbot
COPY supervisord.conf /etc/supervisor/supervisord.conf

CMD /usr/bin/supervisord -n -c /etc/supervisor/supervisord.conf
