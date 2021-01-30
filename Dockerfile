FROM scratch

ADD ./bin/FTPTrap_unix /

ENTRYPOINT [ "/FTPTrap_unix" ]