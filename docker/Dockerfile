FROM debian:jessie

ENV RELEASE_URL=https://github.com/odwrtw/polochon/releases/download/latest/polochon_linux_amd64

RUN apt-get update && \
    apt-get install -y ca-certificates sudo && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/* && \
    adduser polochon

# Download the latest polochon from github's releases
ADD $RELEASE_URL /usr/bin/polochon
RUN chmod +x /usr/bin/polochon

# Expose the http server
EXPOSE 8080

WORKDIR /home/polochon

ADD run.sh /root/run.sh
CMD /root/run.sh
