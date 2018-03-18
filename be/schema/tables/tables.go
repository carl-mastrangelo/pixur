//go:generate protoc tables.proto -I../../../../../ -I. --plugin=pxrtab  --pxrtab_out=. --go_out=.
package tables // import "pixur.org/pixur/be/schema/tables"
