# build stage
FROM golang:1.12 as go

WORKDIR /k8s-namespace-role-binding

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build

# final stage
FROM scratch
COPY --from=go /k8s-namespace-role-binding/k8s-namespace-role-binding /bin/k8s-namespace-role-binding

ENTRYPOINT ["/bin/k8s-namespace-role-binding"]