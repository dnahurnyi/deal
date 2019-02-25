FROM debian:latest
MAINTAINER Denys Nahurnyi dnahurnyi@gmail.com

RUN apt-get update
RUN apt-get install -y ca-certificates

COPY watchersvc /usr/bin/watchersvc
RUN chmod +x /usr/bin/watchersvc
CMD ["watchersvc"]
