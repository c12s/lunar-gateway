FROM golang:latest as builder

WORKDIR /app

COPY ./lunar-gateway/go.mod ./lunar-gateway/go.sum ./

COPY ./oort ../oort
COPY ./magnetar ../magnetar
COPY ./apollo ../apollo
COPY ./heliosphere ../heliosphere

RUN go mod download

COPY ./oort ../oort
COPY ./magnetar ../magnetar
COPY ./apollo ../apollo
COPY ./heliosphere ../heliosphere

COPY ./lunar-gateway .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/main .
COPY --from=builder /app/config/config.yml /root/
COPY --from=builder /app/config/no_auth_config.yml /root/

EXPOSE 5555

# za ukljucivanje sistemskog rl
#CMD ["./main", "sysrl"]

CMD ["./main"]
