# gRPC 

## Introduction

- Make - good and commonly used build tool 

```
Installing Choco

choco install make

```

- Running First Microservice and Frontend

```
From front-end

go run ./cmd/web

```

### Creating First Mioro Service

- Create folder broker-service and run `go mod init broker` this will create a go.mod file
- Create folder within it `cmd/api/main.go`

```
Install chi for routing 

go get github.com/go-chi/chi/v5


```
- Files in `broker-service/cmd`

main.go : 

```go
package main

import (
	"fmt"
	"log"
	"net/http"
)

const webPort = "80"

type Config struct {
}

func main() {

	app := Config{}

	log.Printf("Starting broker service on port %s\n", webPort)

	// define http server

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", webPort),
		Handler: app.routes(),
	}
	// start the server

	err := srv.ListenAndServe()

	if err != nil {
		log.Panic(err)
	}
}

```

routes.go : 

```go
package main

import (
  "github.com/go-chi/chi/v5"
  "github.com/go-chi/chi/v5/middleware"
  "github.com/go-chi/cors"
  "net/http"
)

func (app *Config) routes() http.Handler {
  mux := chi.NewRouter()

  // who is allowed to connect

  mux.Use(cors.Handler(cors.Options{
    AllowedOrigins:   []string{"http://*", "https://*"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "OPTIONS", "DELETE"},
    AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
    ExposedHeaders:   []string{"Link"},
    AllowCredentials: true,
    MaxAge:           300,
  }))

  mux.Use(middleware.Heartbeat("/ping"))

  mux.Post("/", app.Broker)

  return mux
}


```

handlers.go : 

```go
package main

import (
  "encoding/json"
  "net/http"
)

type jsonResponse struct {
  Error   bool   `json:"error"`
  Message string `json:"message"`
  Data    any    `json:"data,omitempty"`
}

func (app *Config) Broker(w http.ResponseWriter, r *http.Request) {
  payload := jsonResponse{
    Error:   false,
    Message: "Hit the broker",
  }
  out, _ := json.MarshalIndent(payload, "", "\t")
  w.Header().Set("Content-Type", "application/json")
  w.WriteHeader(http.StatusAccepted)
  w.Write(out)
}

```


### **Creating a Docker Image for our Docker Image**
- Create a `broker-ssrvice.dockerfile` where the broker code is 
- In the Dockerfile , we are created 2 images- one to build the code and the other is just the executable

```dockerfile
# base go image
FROM golang:1.18-alpine as builder

RUN mkdir /app

COPY . /app

WORKDIR /app

RUN CGO_ENABLED=0 go build -o brokerApp ./cmd/api

RUN chmod +x /app/brokerApp

# build a tiny dokcer image

FROM alpine:latest

RUN mkdir /app

COPY --from=builder /app/brokerApp /app

CMD ["/app/brokerApp"]
```

- The first Docker image code will :
  - Create `/app` on the image
  - Copy all the files where the dockerfile is to the `/app` directory
  - Seting `/app` as the working directory on the container 
  - Build the go app 
  - Make it into an executable

- The second Docker image code will :
  - Create the `app` directory on the machine
  - Copies the executable from the first image to this new image
  - Runs the app 

- cd to `project/docker-compose.yml` and run `docker-compose up -d`

- **Refactoring using Helper Functions to Read JSON and Write JSON**

helpers.go : 

```go
package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type jsonResponse struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (app *Config) readJSON(w http.ResponseWriter, r *http.Request, data any) error {
	maxBytes := 1048576 // 1mb

	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	dec := json.NewDecoder(r.Body)
	err := dec.Decode(data)

	if err != nil {
		return err
	}

	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("body must have single JSON value")
	}

	return nil
}

func (app *Config) writeJSON(w http.ResponseWriter, status int, data any, headers ...http.Header) error {
	out, err := json.Marshal(data)

	if err != nil {
		return err
	}

	if len(headers) > 0 {
		for key, value := range headers[0] {
			w.Header()[key] = value
		}
	}

	w.Header().Set("Content Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(out)
	if err != nil {
		return err
	}
	return nil
}

func (app *Config) errorJSON(w http.ResponseWriter, err error, status ...int) error {
	statusCode := http.StatusBadRequest

	if len(status) > 0 {
		statusCode = status[0]
	}

	var payload jsonResponse
	payload.Error = true
	payload.Message = err.Error()

	return app.writeJSON(w, statusCode, payload)

}


```

### Makefile 

- We can use makefile to run docker steps more easily 
- Ex to build run `make up_build` 

## Authentication Service

- Created **Authentication Service** Folder
- cmd/app/main.go
```go
package main

import (
	"authentication/data"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

const webPort = "80"

var counts int64

type Config struct {
	DB     *sql.DB
	Models data.Models
}

func main() {
	log.Println("Starting Authentication Service")

	app := Config{}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", webPort),
		Handler: app.Routes(),
	}

	err := srv.ListenAndServe()

	if err != nil {
		log.Panic(err)
	}

}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)

	if err != nil {
		return nil, err
	}

	err = db.Ping()

	if err != nil {
		return nil, err
	}

	return db, nil
}

func connectToDB() *sql.DB {

	// this function will try to connect to the DB continuously with a 2s interval
	// It will try 10 times and then fail by returning nil
	
	dsn := os.Getenv("DSN")

	for {
		connection, err := openDB(dsn)

		if err != nil {
			log.Println("Postgres not ready yet...")
			counts++
		} else {
			log.Println("Connected to Postgres")
			return connection
		}

		if counts > 10 {
			log.Println(err)
			return nil
		}

		log.Println("Backing off for 2 seconds")
		time.Sleep(2 * time.Second)
		continue
	}
}

```

- Docker compose for Postgres 

```dockerfile
postgres:
    image: 'postgres:14.2'
    ports:
      - "5432:5432"
    deploy:
      mode: replicated
      replicas: 1
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password
      POSTGRES_DB: users
    volumes:
      - ./db-data/postgres/:/var/lib/postgresql/data/
```

- Note : For Windows , Docker containers run on the WSL Backend and not Localhost. 
- To get address of Postgres DB that docker launches on Port 5432 , we need to type :
- `wsl hostname -I` which gives us the hostname ex: `172.22.138.169`
= We can enter this hostname followed by the port in `pgAdmin` to connect to our DB