# pingwatch
Simple monitoring tool for your Internet connection, pings hosts and reports on availability and trends

## building

Prerequisites:

* Go 1.13 or later

```
git clone https://github.com/fazalmajid/pingwatch
cd pingwatch
go build
```

On Linux, run:

```
sudo setcap cap_net_raw=+ep pingwatch
```

this allows `pingwatch` to open raw sockets so it can send ICMP packets.
