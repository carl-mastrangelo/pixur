//go:generate protoc tables.proto -I../../../../ -I. --plugin=pxrtab  --pxrtab_out=. --go_out=.
package tables
