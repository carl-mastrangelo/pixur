//go:generate /bin/sh -e -c "tail -n +7 ./generate.go | /bin/sh -e"

// Package api contains the Protobuf message used to communicate with the backend server.
package api // import "pixur.org/pixur/api"

/*
sed -i "s/api_version: [0-9]\\+/api_version: `date +%Y%m%d`/" api.proto \
&& protoc api.proto -I. -I../../../ --go_out=plugins=grpc,paths=source_relative:. \
&& protoc data.proto -I. --go_out=plugins=grpc,paths=source_relative:.

exit 0
*/
