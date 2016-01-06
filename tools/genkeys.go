package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log"
	"os"
)

func run() error {
	priv, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return err
	}

	privf, err := os.OpenFile("priv.key", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer privf.Close()

	privblock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv),
	}

	if err := pem.Encode(privf, privblock); err != nil {
		os.Remove(privf.Name())
		return err
	}

	pub, err := x509.MarshalPKIXPublicKey(priv.Public())
	if err != nil {
		os.Remove(privf.Name())
		return err
	}

	pubf, err := os.OpenFile("pub.key", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		os.Remove(privf.Name())
		return err
	}
	defer pubf.Close()

	pubblock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pub,
	}

	if err := pem.Encode(pubf, pubblock); err != nil {
		os.Remove(privf.Name())
		os.Remove(pubf.Name())
		return err
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
