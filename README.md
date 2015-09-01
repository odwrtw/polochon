# polochon

[![Build Status](https://travis-ci.org/odwrtw/polochon.svg?branch=master)](https://travis-ci.org/odwrtw/polochon)
[![Coverage Status](https://coveralls.io/repos/odwrtw/polochon/badge.svg?branch=master&service=github)](https://coveralls.io/github/odwrtw/polochon?branch=master)


## How to use

Copy config.yml.example and customize it as you wish

### Build and launch
```
go build -o polochon server/*.go
./polochon -configPath=/home/user/config.yml
```

### Run
```
go run server/*.go -configPath=/home/user/config.yml
```

## Modules

Polochon was built around modules. They add extensibility and allows us to keep external services out of the main code base. They provide notifications, subtitles and much more. For modules documentation, check this [page](./modules/README.md).
