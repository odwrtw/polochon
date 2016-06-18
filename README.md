# polochon

[![Build Status](https://travis-ci.org/odwrtw/polochon.svg?branch=master)](https://travis-ci.org/odwrtw/polochon)
[![Coverage Status](https://coveralls.io/repos/odwrtw/polochon/badge.svg?branch=master&service=github)](https://coveralls.io/github/odwrtw/polochon?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/odwrtw/polochon)](https://goreportcard.com/report/github.com/odwrtw/polochon)


## How to use

Copy config.yml.example and customize it as you wish

### Build and launch

#### From GitHub release

```
$ curl -L https://github.com/odwrtw/polochon/releases/download/latest/polochon_$(go env GOOS)_$(go env GOARCH) -o polochon
$ chmod +x polochon
$ ./polochon -configPath=/home/user/config.yml
```

#### From source

```
$ make build
$ builds/polochon_$(go env GOOS)_$(go env GOARCH) -configPath=/home/user/config.yml
```

## Modules

Polochon was built around modules. They add extensibility and allows us to keep external services out of the main code base. They provide notifications, subtitles and much more. For modules documentation, check this [page](./modules/README.md).
