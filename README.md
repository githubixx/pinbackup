pinbackup
=========

Backup your Pinterest boards locally. This project is mainly for getting a little bit into Go programming and other technologies. So it's just for fun. While it works for me quite well expect a few rough edges.


Installation
------------

Clone the Git repository:

```
git clone https://github.com/githubixx/pinbackup
cd pinbackup
```

The `master` branch is used for development. While it should be in a good shape in general stable releases are tagged. Check out the latest tag e.g.:

```
# List available tags
git tag

# Checkout latest tag
git checkout 0.1.0
```


Configuration
-------------

Open the **docker-compose.yml** file if you want to change the build settings but it should work with the default values. But there are two important settings that must be adjusted:

```
LOGIN_NAME: "your@email.address"
LOGIN_PASSWORD: "secret"
```

Replace the values with your Pinterest login name and password (Google, Facebook, ... logins don't work). Also of possible interest where the `downloader` stores the images:

```
downloader:
  volumes:
    - /tmp:/tmp
```

The default base path is `/tmp`. The final path of the pictues is composed of base path + user + board name e.g. `/tmp/user/board/`.


Build and run
-------------

The easiest way to build the binary and run it later is using [Docker Compose](https://docs.docker.com/compose/install/). So to build the binary and the container images run

```
docker-compose build
```

To start the containers all the container run

```
docker-compose start -d
```


Usage
-----

To start a board download you need `curl` or `wget`. There is no UI (yet). So if you want to download the pictures of `https://www.pinterest.com/user/board/` e.g. use this command:

```
curl --header "Content-Type: application/json" \
     --request POST \
     --data '{"url": "https://www.pinterest.com/user/board/"}'
     http://localhost:8080/api/v1/board
```

Of course you can also use national domains here too like `pinterest.de` and so on. This returns a JSON response which includes a `uuid`. To pretty print the output you can use the `jq` utility by adding a pipe (`| jq` e.g.).

To stop the containers use

```
docker-compose stop
```
