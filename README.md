# shortr

***Blazingly fast*** url shortener, written in go

## Instaallation
The first step is cloning this repo:

```shell
git clone https://github.com/4rkal/shortr && cd shortr
```

Edit the `docker-compose.yml` and set a base url:

    command: ["/shortr", "--url", "YOUR_URL"]
    
Change the `YOUR_URL`, to the url where your app served from (eg app.4rkal.com)

Now you can run the application via docker.

### Using docker
Assuming that you have docker compose installed simply run

```
docker compose up
```

Now the application should be available by vissiting `localhost:8080`

