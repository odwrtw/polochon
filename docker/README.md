## Build

```
cp docker-compose.yml.example docker-compose.yml
docker-compose up -d
```

## Environment variables

* $POLOCHON_CONFIG => polochon configuration file
* $POLOCHON_TOKEN => token manager configuration file
* $UID => polochon's user UID
* $GID => polochon's user GID

## Run

```
docker-compose run
```

## Limitation

The files must all be mounted under the same docker volume. If you use separate volumes, polochon will rise a "invalid cross-device link" error while trying to move the file from one volume to another.
