# syntax=docker/dockerfile:1

FROM golang:1.23
WORKDIR /shopping-list-server

# Copying the go.mod and go.sum files into the images to get ready for compilation
COPY go.mod ./
# Now download the dependencies
RUN go mod download
# Copy the src files into the image
# COPY *.go ./
# Copy the other directories as well
COPY internal ./internal
COPY setup ./setup
COPY cmd/shopping-list-server/main.go ./cmd/shopping-list-server/main.go

# DEBUG: Write the output of ls into a file
# RUN echo "$(ls -1 /shopping-list)"

# Compile the application
RUN CGO_ENABLED=0 GOOS=linux go build -o ./app-server ./cmd/shopping-list-server

# Copy the configuration
COPY resources/dockerDb.json ./resources/db.json
COPY resources/jwtSecret.json ./resources/jwtSecret.json
COPY resources/shoppinglist.crt ./resources/shoppinglist.crt
COPY resources/shoppinglist.pem ./resources/shoppinglist.pem
COPY resources/whitelisted_ips.json ./resources/whitelisted_ips.json

# Starting the go application
CMD [ "/app-server" ]