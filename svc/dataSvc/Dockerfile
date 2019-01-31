FROM debian:latest
MAINTAINER Denys Nahurnyi dnahurnyi@gmail.com

RUN apt-get update
RUN apt-get install -y ca-certificates

COPY datasvc /usr/bin/datasvc
RUN chmod +x /usr/bin/datasvc
CMD ["datasvc"]
