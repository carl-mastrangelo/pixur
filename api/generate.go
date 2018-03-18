//go:generate /bin/sh -e -c "tail -n +5 ./generate.go | /bin/sh -e"
package api

/*
sed -i "s/api_version: [0-9]\\+/api_version: `date +%Y%m%d`/" api.proto \
&& protoc api.proto --go_out=plugins=grpc:.

exit 0
*/
