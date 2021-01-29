package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

//Config ..
type Config struct {
	Users     map[string]string //Blank strings all all passwords for username
	JunkFiles map[string]JunkFile
}

var junkConfig Config

func loadConfig(path string) error {
	jsonFile, err := os.Open(path)
	if err == nil {
		defer jsonFile.Close()
		byteValue, _ := ioutil.ReadAll(jsonFile)
		err = json.Unmarshal(byteValue, &junkConfig)
	}
	return err
}

var (
	noauth       bool
	keyPath      string
	configPath   string
	port         string
	loginDelay   int64
	commandDelay int64
)

func main() {

	flag.BoolVar(&noauth, "no-auth", false, "no authentication")
	flag.StringVar(&keyPath, "key-path", "test.key", "Path to key.  Defaults to test.key")
	flag.StringVar(&configPath, "config-path", "config.json", "Path to config file.")
	flag.StringVar(&port, "port", "2022", "Port to run ftp on")
	flag.Int64Var(&loginDelay, "login-delay", 0, "How long to delay login attempts in seconds")
	flag.Int64Var(&commandDelay, "command-delay", 0, "How long to delay commands in seconds")

	flag.Parse()

	if err := loadConfig(configPath); err != nil {
		fmt.Println("Failed loading config", err)
	}

	config := &ssh.ServerConfig{}

	//Enable or disable auth based on runtime flag
	if noauth {
		config.NoClientAuth = true
	} else {
		config.PasswordCallback = func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			fmt.Println("Login attempt from:" + c.RemoteAddr().String() + " " + c.RemoteAddr().Network())
			if loginDelay > 0 {
				time.Sleep(time.Second * time.Duration(loginDelay))
			}
			//Look up users from config
			password, ok := junkConfig.Users[c.User()]
			fmt.Println(c.User(), ":", password)
			if ok {
				//If the password in the config is blank just let anyone in.
				if string(pass) == password || password == "" {
					return nil, nil
				}
			}

			return nil, fmt.Errorf("password rejected for %q", c.User())
		}
	}

	privateBytes, err := ioutil.ReadFile(keyPath)
	if err != nil {
		log.Fatal("Failed to load private key", err)
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		log.Fatal("Failed to parse private key", err)
	}

	config.AddHostKey(private)

	listener, err := net.Listen("tcp", "0.0.0.0:"+port)
	if err != nil {
		log.Fatal("failed to listen for connection", err)
	}
	fmt.Printf("Listening on %v\n", listener.Addr())

	for {
		nConn, err := listener.Accept()
		if err != nil {
			log.Print("failed to accept incoming connection", err)
		}

		go HandleConnection(nConn, config)
	}
}

//HandleConnection handle the incoming connection
func HandleConnection(nConn net.Conn, config *ssh.ServerConfig) {
	// Before use, a handshake must be performed on the incoming
	sConn, chans, reqs, err := ssh.NewServerConn(nConn, config)
	fmt.Println("Connection from:" + nConn.RemoteAddr().String())
	if err != nil {
		log.Print("Handshake failed", err)
		return
	}

	// The incoming Request channel must be serviced.
	go ssh.DiscardRequests(reqs)

	// Service the incoming Channel channel.
	for newChannel := range chans {
		// We only want a session.
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			log.Printf("unknown channel type: %s", newChannel.ChannelType())
			continue
		}
		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Print("could not accept channel")
			continue
		}

		// Sessions have out-of-band requests such as "shell",
		// "pty-req" and "env".  Here we handle only the
		// "subsystem" request.
		go func(in <-chan *ssh.Request) {
			for req := range in {
				ok := false
				switch req.Type {
				case "subsystem":
					if string(req.Payload[4:]) == "sftp" {
						ok = true
					}
				}
				req.Reply(ok, nil)
			}
		}(requests)

		//Setup request server for the incoming connection
		handlers, err := GetJunkHandler(sConn.User(), commandDelay)
		if err != nil {
			continue
		}

		server := sftp.NewRequestServer(
			channel,
			handlers,
		)

		if err := server.Serve(); err == io.EOF {
			server.Close()
		} else if err != nil {
			server.Close()
			log.Print("SFTP server completed with error")
		}
	}
}
