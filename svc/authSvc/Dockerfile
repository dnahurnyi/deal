FROM debian:latest
MAINTAINER Denys Nahurnyi dnahurnyi@gmail.com

RUN apt-get update
RUN apt-get install -y ca-certificates

COPY authsvc /usr/bin/authsvc
COPY private.pem /usr/bin/private.pem
COPY public.pem /usr/bin/public.pem
RUN chmod 777 /usr/bin/private.pem
RUN chmod 777 /usr/bin/public.pem
RUN chmod +x /usr/bin/authsvc
CMD ["authsvc"]
