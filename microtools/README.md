## Cmd

## Config
## Selector
### Note on Linux Support:
This library attempts to send an "unprivileged" ping via UDP. On linux, this must be enabled by setting
```shell script
sudo sysctl -w net.ipv4.ping_group_range="0   2147483647"
```

If you do not wish to do this, you can set pinger.SetPrivileged(true) and use setcap to allow your binary using go-ping to bind to raw sockets (or just run as super-user):

```shell script
setcap cap_net_raw=+ep /bin/go-ping
```