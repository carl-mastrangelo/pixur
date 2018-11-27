//go:generate protoc tables.proto -I../../../../../ -I. --plugin=pxrtab  --pxrtab_out=. --go_out=paths=source_relative:.

// Package tables includes all generated tables for Pixur.
package tables // import "pixur.org/pixur/be/schema/tables"
