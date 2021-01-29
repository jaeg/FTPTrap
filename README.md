# FTPTrap

Keep the brute force ftp bots busy with this simple waste of time. 

## Features
- Allows for specifying fake file names and their contents. 
- Allows for specifing a no auth mode to allow all connections to access the server.
- Allows for specifying a set of fake users with the option of leaving the password blank to allow all passwords. 
- All upload commands and file change commands are ignored.
- Can purposely slow login attempts and commands

## Flags
- no-auth : Disable authentication into the FTPTrap
- key-path : Path to private key
- config-path : Path to config.json
- port : Port to run the FTP server on
- login-delay : How long in seconds to delay login attempts
- command-delay : How long in seconds to delay all commands