# polochon

[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://pkg.go.dev/github.com/odwrtw/polochon/lib)
[![Build Status](https://github.com/odwrtw/polochon/workflows/Build/badge.svg)](https://github.com/odwrtw/polochon/actions)
[![Coverage Status](https://coveralls.io/repos/odwrtw/polochon/badge.svg?branch=master&service=github)](https://coveralls.io/github/odwrtw/polochon?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/odwrtw/polochon)](https://goreportcard.com/report/github.com/odwrtw/polochon)


## How to use

There are two configuration files required for this application to work properly:
* The main configuration file is heavily commented to explain each configuration option.
* The token configuration file is here to give a fine grain control over the rights of the HTTP server.

To get started, simply copy those files and customise them as needed.

```
cp config.example.yml config.yml
cp token.exemple.yml token.yml
```

### Build and launch

#### From GitHub release

```sh
curl -L https://github.com/odwrtw/polochon/releases/download/latest/polochon_$(go env GOOS)_$(go env GOARCH) -o polochon
chmod +x polochon
./polochon -configPath=/home/user/config.yml -tokenPath=/home/user/token.yml
```

#### From source

```sh
cd app
go build *.go
```

#### From a docker image

The docker images is built on the docker hub and can be downloaded using the following command:

```sh
docker pull odwrtw/polochon
```

A `docker-compose` file example is available [here](./docker/docker-compose.yml.example) and the readme for this file [here](./docker/README.md).
