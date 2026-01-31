FROM alpine:3.20

WORKDIR /app

COPY ./bin/consigliere_linux_amd64 /app/consigliere

CMD ["./consigliere"]
