Pixur
=====

Pixur is a picture management server and frontend.  It is designed to support
a loose community of users.

Features:

* Pixur has strong access controls.  It is possible to restrict access to all 
  content and functionality.
* Design to scale.  The intended goal is to be able to store about 100 million
  pictures in a single installation.
* Weak / Strong deletion semantics.  Pictures have well defined life times, and 
  can be partially or weakly deleted.  This allows the picuture metadata to 
  remain while removing the file content.
* Individual picture recommendation and scoring.  Every user gets customized
  picture recommendations based on previous voting behavior
* High accessibility.  Pixur allows access over HTTP, WebDAV, gRPC, etc.

## Installation

1.  Get the main Pixur server, and site initializer.
```
go get -u pixur.org/pixur{,/tools/initsite}
```

2.  Create initial configuration files.  The `initsite` tool 
will prompt you to put in the necessary information.

```
go run $GOPATH/src/pixur.org/pixur/tools/initsite/initsite.go
``` 

3.  Start the Pixur Server.

```
go run $GOPATH/src/pixur.org/pixur/pixur.go
```


## Requirements

* [ffmpeg](https://www.ffmpeg.org/) is needed to handle WEBM content.
* One of MySQL, SQLite3, PostgreSQL, or CockroachDB is needed for data storage
 

