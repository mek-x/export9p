package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh/terminal"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "  %s [options] user [password]\n", os.Args[0])
		flag.PrintDefaults()
	}
	count := flag.Int("c", 12, "Count value used in bcrypt algo")
	out := flag.String("o", "", "Name of the file for appending the passwords. If not given, uses stdout")
	flag.Parse()

	user := flag.Arg(0)
	pass := flag.Arg(1)

	if user == "" {
		flag.Usage()
		log.Fatalln("User needs to be given")
	}

	if pass == "" {
		fmt.Print("Enter password:")
		p, _ := terminal.ReadPassword(int(os.Stdin.Fd()))
		fmt.Print("\nVerify password:")
		p2, _ := terminal.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if string(p) != string(p2) {
			log.Fatalln("Passwords don't match")
		}
		pass = string(p)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(pass), *count)
	if err != nil {
		log.Fatal(err)
	}

	line := fmt.Sprintf("%s:%s\n", user, string(hash))

	if *out != "" {
		f, err := os.OpenFile(*out, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		f.WriteString(line)
	} else {
		fmt.Print(line)
	}
}
