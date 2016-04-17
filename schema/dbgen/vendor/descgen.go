//go:generate /bin/sh -e -c "tail -n +5 ./descgen.go | /bin/sh -e"
package _

/*
DESCRIPTOR_PATH="$(dirname `which protoc` | sed 's/bin/include/')/google/protobuf/descriptor.proto"
protoc $DESCRIPTOR_PATH --go_out=. &&

exit 0
*/
