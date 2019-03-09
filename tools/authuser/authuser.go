// Package authuser fetches an auth token for command-line use.
package main // import "pixur.org/pixur/tools/authuser"

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
	"google.golang.org/grpc"

	"pixur.org/pixur/api"
)

// Sample usage:
// go run authuser.go '--spec=dns:///localhost:8889' > ~/.pxrtoken.pb.txt

var flagSpec = flag.String("spec", "", "The Pixur gRPC server address")

func run(ctx context.Context, spec string) error {
	ch, err := grpc.DialContext(ctx, spec, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer ch.Close()
	client := api.NewPixurServiceClient(ch)

	fmt.Fprint(os.Stderr, "User Ident (e.g. foo@bar.com): ")
	data, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return err
	}
	ident := string(data[:len(data)-1])

	fmt.Fprint(os.Stderr, "Admin Secret Password (e.g. 12345): ")
	secret, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return err
	}

	res, err := client.GetRefreshToken(ctx, &api.GetRefreshTokenRequest{
		Ident:  ident,
		Secret: string(secret),
	})
	if err != nil {
		return err
	}

	fmt.Fprint(os.Stdout, res)
	fmt.Fprint(os.Stderr, "\nSuccess!\n")
	return nil
}

func main() {
	flag.Parse()

	if err := run(context.Background(), *flagSpec); err != nil {
		panic(err)
	}
}
