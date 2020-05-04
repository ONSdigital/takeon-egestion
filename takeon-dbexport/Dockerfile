FROM golang:1.13.4
WORKDIR /go/src/app
COPY . .
RUN go get -d -v ./...
RUN go get -u github.com/aws/aws-lambda-go/lambda
RUN go install -v ./...
CMD ["app"]
RUN apt-get update -y &&\
    apt-get install curl -y &&\
    apt-get install awscli -y &&\
    curl -sL https://deb.nodesource.com/setup_10.x | bash &&\
    apt-get install nodejs -y &&\
    node -v &&\
    npm -v &&\
    npm install -g serverless@1.52.2


# FROM debian:stable-slim

# RUN apt-get update -y &&\
# 	  apt-get install jq -y &&\
#     apt-get install curl -y
# RUN apt-get update -y &&\
#     apt-get install awscli -y &&\
#     curl -sL https://deb.nodesource.com/setup_10.x | bash &&\
#     apt-get install nodejs -y &&\
#     node -v &&\
#     npm -v &&\
#     npm install -g serverless