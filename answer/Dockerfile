FROM golang:1.21

COPY ./answer.go /answer.go

RUN cd / && go mod init main && go get .

ENTRYPOINT ["go", "run", "/answer.go"]

EXPOSE 60000
