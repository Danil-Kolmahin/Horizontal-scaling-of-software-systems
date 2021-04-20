FROM golang:1.15 as build

RUN apt-get update && apt-get install -y ninja-build

RUN go get -u github.com/BohdanShmalko/Implementation-assembly-system/build/cmd/bood

WORKDIR /go/src/practice-2
COPY . .

RUN CGO_ENABLED=0 bood

# ==== Final image ====
FROM ubuntu
WORKDIR /opt/practice-2
COPY entry.sh ./
COPY --from=build /go/src/practice-2/out/bin/* ./
RUN ls
ENTRYPOINT ["/opt/practice-2/entry.sh"]
CMD ["server"]
