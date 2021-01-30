package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

//Config ..
type Config struct {
	Users        map[string]string //Blank strings all all passwords for username
	JunkFiles    map[string]JunkFile
	CommandDelay time.Duration
	LoginDelay   time.Duration
}

var junkConfig Config

type activityEntry struct {
	user      string
	ip        string
	action    string
	timestamp time.Time
}

var activity []activityEntry
var activityMu sync.Mutex

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
	activity = make([]activityEntry, 0)

	flag.BoolVar(&noauth, "no-auth", false, "no authentication")
	flag.StringVar(&keyPath, "key-path", "", "Path to key if set, otherwise generate cert.")
	flag.StringVar(&configPath, "config-path", "config.json", "Path to config file.")
	flag.StringVar(&port, "port", "2022", "Port to run ftp on")
	flag.Int64Var(&loginDelay, "login-delay", -1, "How long to delay login attempts in seconds")
	flag.Int64Var(&commandDelay, "command-delay", -1, "How long to delay commands in seconds")

	flag.Parse()

	if err := loadConfig(configPath); err != nil {
		fmt.Println("Failed loading config", err)
	}

	if loginDelay >= 0 {
		junkConfig.LoginDelay = time.Duration(loginDelay)
	}
	if commandDelay >= 0 {
		junkConfig.CommandDelay = time.Duration(commandDelay)
	}

	config := &ssh.ServerConfig{}

	//Enable or disable auth based on runtime flag
	if noauth {
		config.NoClientAuth = true
	} else {
		config.PasswordCallback = func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			logActivity(c.RemoteAddr().String(), c.User(), "Login Attempt: "+string(pass))
			if junkConfig.LoginDelay > 0 {
				time.Sleep(time.Second * junkConfig.LoginDelay)
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
			logActivity(c.RemoteAddr().String(), c.User(), "Login Rejected: "+string(pass))

			return nil, fmt.Errorf("password rejected for %q", c.User())
		}
	}

	if keyPath != "" {
		privateBytes, err := ioutil.ReadFile(keyPath)
		if err != nil {
			log.Fatal("Failed to load private key", err)
		}

		private, err := ssh.ParsePrivateKey(privateBytes)
		if err != nil {
			log.Fatal("Failed to parse private key", err)
		}

		config.AddHostKey(private)
	} else {
		log.Println("No key supplied, generating one now..")
		caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			log.Fatal("Failed to create private key")
		}
		certPrivKeyPEM := new(bytes.Buffer)
		pem.Encode(certPrivKeyPEM, &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
		})
		private, err := ssh.ParsePrivateKey(certPrivKeyPEM.Bytes())
		if err != nil {
			log.Fatal("Failed to parse private key", err)
		}

		log.Println("Finished generating key")

		config.AddHostKey(private)
	}

	listener, err := net.Listen("tcp", "0.0.0.0:"+port)
	if err != nil {
		log.Fatal("failed to listen for connection", err)
	}
	fmt.Printf("Listening on %v\n", listener.Addr())

	go writeActivityToDisk()
	for {
		nConn, err := listener.Accept()
		if err != nil {
			log.Println("failed to accept incoming connection", err)
		}

		go HandleConnection(nConn, config)
	}
}

// We don't want attackers to be able to abuse our disk IO by DOSing us with activity.
// The idea is that this process will handle getting that activity logged on its own schedule.
func writeActivityToDisk() {
	for {
		file, err := os.OpenFile("activity.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err == nil {
			if len(activity) > 0 {
				activityMu.Lock()
				//Write activity to disk.
				for _, ac := range activity {
					file.WriteString(ac.timestamp.String() + " - IP:" + ac.ip + " USER: " + ac.user + " ACTION:" + ac.action + "\n")
				}

				//Empty activity array.
				activity = make([]activityEntry, 0)
				activityMu.Unlock()
			}
			file.Close()
		} else {
			log.Fatal(err)
		}

		time.Sleep(5 * time.Second)
	}
}

//HandleConnection handle the incoming connection
func HandleConnection(nConn net.Conn, config *ssh.ServerConfig) {
	// Before use, a handshake must be performed on the incoming
	sConn, chans, reqs, err := ssh.NewServerConn(nConn, config)
	fmt.Println("Connection from:" + nConn.RemoteAddr().String())
	if err != nil {
		log.Println("Handshake failed", err)
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
			log.Println("could not accept channel")
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
		handlers, err := GetJunkHandler(sConn.User(), sConn.RemoteAddr().String())
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
			log.Println("SFTP server completed with error")
		}
	}
}

func logActivity(ip string, user string, action string) {
	ac := activityEntry{ip: ip, user: user, action: action, timestamp: time.Now()}
	activityMu.Lock()
	activity = append(activity, ac)
	activityMu.Unlock()
	fmt.Println(ac.timestamp.String() + " " + ac.ip + ":" + ac.user + " - " + ac.action)
}
