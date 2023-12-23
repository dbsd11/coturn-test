FROM golang:1.21

COPY ./offer.go /offer.go

RUN cd / && go mod init main && go get .

ENTRYPOINT ["go", "run", "/offer.go"]

EXPOSE 50000
