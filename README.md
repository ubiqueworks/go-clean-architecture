### Supporting code for [Go Clean](https://medium.com/@teo2k/go-clean-54c5cd866fe5) article on Medium

#### Required software:
* Docker and Docker Compose
* Dep
* Bazel
* GoImports

#### Running the project (tested on Mac OSX)

Make sure GOPATH is set and clone into the correct directory under GOPATH
```
git clone https://github.com/ubiqueworks/go-clean-architecture.git $GOPATH/src/src/github.com/ubiqueworks/go-clean-architecture
```

Go to the root directory of the project.
```
cd $GOPATH/src/src/github.com/ubiqueworks/go-clean-architecture
```

Ensure dependencies

```
dep ensure
```

Build the docker images

```
make package
```

Run the containers

```
docker-compose up
```

Delete the containers

```
docker-compose down -v
```

#### Publish a message
```
curl -X "POST" "http://localhost:8888/publish" \
     -H 'Content-Type: application/json; charset=utf-8' \
     -d $'{
  "name": "Random Name",
  "message": "Necessitatibus magnam animi magnam fuga nihil soluta est quis. Quo dolor sit sit ut quia aspernatur. Porro ut dolores consequatur optio harum et laborum magni. Illum incidunt amet molestias quo vitae. Inventore eos ut dolores deserunt. Ut error ut in est et temporibus."
}'
```

#### Get all published messages
```
curl http://localhost:8888/messages | json_pp
```
