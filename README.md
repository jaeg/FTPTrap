# FTPTrap

Keep the brute force ftp bots busy with this simple waste of time. 

## Features
- Allows for specifying fake file names and their contents. 
- Allows for specifing a no auth mode to allow all connections to access the server.
- Allows for specifying a set of fake users with the option of leaving the password blank to allow all passwords. 
- All upload commands and file change commands are ignored.
- Can purposely slow login attempts and commands
- Provides log files for users that attempted to log in as well as command history.

## Flags
- no-auth : Disable authentication into the FTPTrap
- key-path : Path to private key
- config-path : Path to config.json
- port : Port to run the FTP server on
- login-delay : How long in seconds to delay login attempts
- command-delay : How long in seconds to delay all commands
- user-output : Where to output the users tracking information.

## Configuration Examples
Basic:
```{
    "LoginDelay":1,
    "CommandDelay":1,
    "Users": {
        "admin":"password",
        "nopass":""
    },
    "JunkFiles": {
        "/test.txt":{
            "FileName":"Secrets.txt",
            "Content":"gotcha"
        },
        "/test2.txt":{
            "FileName":"id_rsa",
            "Content":"Nope"
        },
        "/test3.txt":{
            "FileName":"passwords.txt",
            "Content":"Password, 1234"
        }
    }
}
```

Comprimised Credential Catcher:
```{
    "LoginDelay":3,
    "CommandDelay":3,
    "Users": {
    },
    "JunkFiles": {
    }
}
```
With no users nothing will be able to authenticate.  Each login attempt is logged and with login delay set to 3 it'll eat up some process time of the attacker.

## Dependencies
- Golang 

## How to run
Run as exectuable:
./FTPTrap_unix --login-delay 3

Run as docker container:
`docker run -v local/path/to/config.json:/config.json -p 2022:2022 jaeg/ftptrap:latest --login-delay 3 --command-delay 3`

## How to build
`make build` will output an executable for your system to the ./bin folder.  
`make build-linux` will generate a linux executable no matter your system to the ./bin folder.
`make image` will build and can a docker image

