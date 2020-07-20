FROM golang:1.14 as build

WORKDIR $GOPATH/src/app
COPY . .

# Fetch dependencies & install utilities
RUN go get -d -v &&\
    apt-get update -y &&\
    apt-get install zip unzip -qy

# Build the executable and package as a zip file
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o bin/main &&\
    zip /lambda.zip bin/main

FROM scratch
COPY --from=build /lambda.zip /





# FROM golang:1.13.4
# WORKDIR /go/src/app
# COPY . .
# RUN go get -d -v ./...
# RUN go get -u github.com/aws/aws-lambda-go/lambda
# RUN go install -v ./...
# CMD ["app"]
# RUN apt-get update -y &&\
#     apt-get install jq -y &&\
#     apt-get install curl -y
# RUN apt-get update -y &&\
#     apt-get install awscli -y &&\
#     curl -sL https://deb.nodesource.com/setup_10.x | bash &&\
#     apt-get install nodejs -y &&\
#     node -v &&\
#     npm -v &&\
#     npm install -g serverless@1.52.2