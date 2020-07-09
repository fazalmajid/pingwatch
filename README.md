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

## Database

The list of hostnames or IPs to ping is in the table `dests`.

To add a destination, run:

```
sqlite3 pingwatch.sqlite << EOF
insert into dests values ('1.1.1.1'), ('8.8.8.8');
EOF
```

The actual measurements are in the table `pings`:

* *time* timestamp in SQLite julian day format, UTC)
* *host* hostname that was pinged
* *ip* IPv4 or IPv6 address *host* resolved to at ping time
* *rtt* ping round-trip time in milliseconds
