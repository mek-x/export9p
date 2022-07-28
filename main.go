package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/emersion/go-sasl"
	"github.com/knusbaum/go9p"
	"github.com/knusbaum/go9p/fs"
	"github.com/knusbaum/go9p/fs/real"
)

var exportFS fs.FS

func PlainAuth(userpass map[string]string) func(io.ReadWriter) (string, error) {
	return func(s io.ReadWriter) (string, error) {
		auth := sasl.NewPlainServer(func(identity, username, password string) error {
			if identity != username {
				return fmt.Errorf("access denied")
			}
			pass, ok := userpass[username]
			if !ok {
				return fmt.Errorf("access denied")
			}
			if bcrypt.CompareHashAndPassword([]byte(pass), []byte(password)) != nil {
				return fmt.Errorf("access denied")
			}
			return nil
		})

		for {
			var ba [4096]byte
			log.Printf("READ1\n")
			n, err := s.Read(ba[:])
			if err != nil {
				return "", err
			}
			bs := ba[:n]
			challenge, done, err := auth.Next(bs)
			if err != nil {
				log.Printf("ERROR: %s\n", err)
				return "", err
			}
			if done {
				parts := bytes.Split(bs, []byte("\x00"))
				log.Printf("SUCCESS!\n")
				return string(parts[0]), nil
			}
			log.Printf("WRITE1\n")
			s.Write(challenge)
		}
	}
}

func main() {
	directory := flag.String("dir", ".", "The directory that will be exported")
	address := flag.String("address", "0.0.0.0:14672", "The address on which to listed for incoming 9p connections")
	srv := flag.String("srv", "", "If specified, exportfs will listen on a unix socket with this service name in the current namespace (see p9p namespace(1)) rather than listening on tcp")
	verbose := flag.Bool("v", false, "Makes the 9p protocol verbose, printing all incoming and outgoing messages.")
	stdio := flag.Bool("s", false, "Serve 9p over standard in and standard out.")
	noperm := flag.Bool("noperm", false, "Ignore permissions enforcement. Any attached user will have the same filesystem permissions as the user running export9p.")
	passwdFile := flag.String("p", "passwd", "Read password hashes from given file")
	flag.Parse()

	if flag.NArg() > 0 {
		log.Printf("Extraneous arguments.")
		flag.Usage()
		os.Exit(1)
	}

	go9p.Verbose = *verbose

	dir, err := filepath.Abs(*directory)
	if err != nil {
		log.Printf("Error: %s", dir)
		flag.Usage()
		os.Exit(1)
	}

	contents, err := os.ReadFile(*passwdFile)
	if err != nil {
		log.Fatal(err)
	}

	userPass := make(map[string]string)

	lines := strings.Split(string(contents), "\n")
	for _, v := range lines {
		pair := strings.Split(v, ":")
		if len(pair) == 2 {
			userPass[pair[0]] = pair[1]
		}
	}

	exportFS.Root = &real.Dir{Path: dir}
	fs.WithCreateFile(real.CreateFile)(&exportFS)
	fs.WithCreateDir(real.CreateDir)(&exportFS)
	fs.WithRemoveFile(real.Remove)(&exportFS)
	fs.WithAuth(PlainAuth(userPass))(&exportFS)
	if *noperm {
		fs.IgnorePermissions()(&exportFS)
	}
	if *stdio {
		if *verbose {
			log.Printf("Serving %s on standard input/output", dir)
		}
		err = go9p.ServeReadWriter(os.Stdin, os.Stdout, exportFS.Server())
	} else if *srv != "" {
		if *verbose {
			log.Printf("Serving %s as service %s", dir, *srv)
		}
		err = go9p.PostSrv(*srv, exportFS.Server())
	} else {
		if *verbose {
			log.Printf("Serving %s on %s", dir, *address)
		}
		err = go9p.Serve(*address, exportFS.Server())
	}
	if err != nil {
		log.Fatal(err)
	}
}
