package main

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net"
	"sync"
	"syscall"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	szLen            = 8
	tsLen            = 8
	protocolICMP     = 1
	protocolIPv6ICMP = 58
)

var (
	ipv4Proto = map[string]string{"ip": "ip4:icmp", "udp": "udp4"}
	ipv6Proto = map[string]string{"ip": "ip6:ipv6-icmp", "udp": "udp6"}
	pingers   map[string]*Pinger
)

type in_flight struct {
	timer   *time.Timer
	pinger  *Pinger
	results chan *Result
	Result
}

type Pinger struct {
	Host string
	// Interval is the wait time between each packet send. Default is 1s.
	Interval time.Duration

	// ICMP timeout
	Timeout time.Duration

	// Size of packet being sent
	Size int

	// stop chan bool
	done chan bool

	conn_ipv4 *icmp.PacketConn
	done_ipv4 chan bool
	conn_ipv6 *icmp.PacketConn
	done_ipv6 chan bool
	recv      chan *packet

	size     int
	id       int
	sequence int
	network  string
	wg       sync.WaitGroup

	pending map[int]in_flight // sequence -> start time
	results chan *Result
}

func NewPinger(host string, interval time.Duration, results chan *Result) (*Pinger, error) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &Pinger{
		Host:      host,
		Interval:  interval,
		Timeout:   5 * time.Second,
		id:        r.Intn(math.MaxInt16),
		network:   "udp",
		Size:      128,
		done:      make(chan bool),
		done_ipv4: make(chan bool),
		done_ipv6: make(chan bool),
		pending:   make(map[int]in_flight, 1),
		results:   results,
	}, nil
}

type packet struct {
	ipv4   bool
	bytes  []byte
	nbytes int
	ttl    int
}

type Result = struct {
	TS   time.Time
	Host string
	IP   string
	Rtt  time.Duration
}

// SetPrivileged sets the type of ping pinger will send.
// false means pinger will send an "unprivileged" UDP ping.
// true means pinger will send a "privileged" raw ICMP ping.
// NOTE: setting to true requires that it be run with super-user privileges.
func (p *Pinger) SetPrivileged(privileged bool) {
	if privileged {
		p.network = "ip"
	} else {
		p.network = "udp"
	}
}

// Privileged returns whether pinger is running in privileged mode.
func (p *Pinger) Privileged() bool {
	return p.network == "ip"
}

func (p *Pinger) start_listener(use_ipv4 bool) (conn *icmp.PacketConn) {
	if use_ipv4 {
		conn = p.listen(ipv4Proto[p.network])
		if conn == nil {
			log.Fatal("could not start IPv4 listener")
		}
		conn.IPv4PacketConn().SetControlMessage(ipv4.FlagTTL, true)
	} else {
		if conn = p.listen(ipv6Proto[p.network]); conn == nil {
			log.Fatal("could not start IPv6 listener")
		}
		conn.IPv6PacketConn().SetControlMessage(ipv6.FlagHopLimit, true)
	}
	go p.recvICMP(conn, use_ipv4)
	return conn
}

func (p *Pinger) recvICMP(conn *icmp.PacketConn, use_ipv4 bool) {
	defer conn.Close()
	defer p.wg.Done()
	var done chan bool

	if use_ipv4 {
		done = p.done_ipv4
	} else {
		done = p.done_ipv6
	}

	for {
		select {
		case <-done:
			return
		default:
			bytes := make([]byte, 512)
			conn.SetReadDeadline(time.Now().Add(time.Millisecond * 100))
			var n, ttl int
			var err error
			if use_ipv4 {
				var cm *ipv4.ControlMessage
				n, cm, _, err = conn.IPv4PacketConn().ReadFrom(bytes)
				if cm != nil {
					ttl = cm.TTL
				}
			} else {
				var cm *ipv6.ControlMessage
				n, cm, _, err = conn.IPv6PacketConn().ReadFrom(bytes)
				if cm != nil {
					ttl = cm.HopLimit
				}
			}
			if err != nil {
				if neterr, ok := err.(*net.OpError); ok {
					if neterr.Timeout() {
						// Read timeout
						continue
					} else {
						close(p.done)
						return
					}
				}
			}

			p.recv <- &packet{
				ipv4:   use_ipv4,
				bytes:  bytes,
				nbytes: n,
				ttl:    ttl,
			}
		}
	}
}

func (p *Pinger) Run() {
	p.recv = make(chan *packet, 5)
	defer close(p.recv)
	p.wg.Add(1)

	if *verbose {
		log.Println("starting pinger", p.Host, "interval", p.Interval)
	}

	err := p.sendICMP()
	if err != nil {
		fmt.Println(err.Error())
	}

	interval := time.NewTicker(p.Interval)
	defer interval.Stop()

	for {
		select {
		case <-p.done:
			p.Stop()
			return
		case <-interval.C:
			err = p.sendICMP()
			if err != nil {
				fmt.Println("FATAL: ", err.Error())
			}
		case r := <-p.recv:
			err := p.processPacket(r, p.results)
			if err != nil {
				fmt.Println("FATAL: ", err.Error())
			}
		}
	}
}

func (p *Pinger) Stop() {
	close(p.done)
	p.done_ipv4 <- true
	p.done_ipv6 <- true
	p.wg.Wait()
}

func (p *Pinger) sendICMP() error {
	var typ icmp.Type
	var conn *icmp.PacketConn

	// this lookup could take longer than the actual ping itself
	ipaddr, err := net.ResolveIPAddr("ip", p.Host)
	if err != nil {
		return err
	}
	addr_string := ipaddr.String()

	if isIPv4(ipaddr.IP) {
		if p.conn_ipv4 == nil {
			p.conn_ipv4 = p.start_listener(true)
		}
		conn = p.conn_ipv4
		typ = ipv4.ICMPTypeEcho
	} else if isIPv6(ipaddr.IP) {
		if p.conn_ipv6 == nil {
			p.conn_ipv6 = p.start_listener(false)
		}
		conn = p.conn_ipv6
		typ = ipv6.ICMPTypeEchoRequest
	}

	var dst net.Addr = ipaddr
	if p.network == "udp" {
		dst = &net.UDPAddr{IP: ipaddr.IP, Zone: ipaddr.Zone}
	}

	now := time.Now()
	payload := bytes.Join(
		[][]byte{
			timeToBytes(now),
			intToBytes(int64(len(p.Host))),
			[]byte(p.Host),
			intToBytes(int64(len(addr_string))),
			[]byte(addr_string),
		},
		[]byte{},
	)
	padding := p.Size - szLen - len(payload)
	if padding > 0 {
		payload = append(payload, bytes.Repeat([]byte{1}, padding)...)
	}
	payload = append(intToBytes(int64(len(payload))), payload...)

	body := &icmp.Echo{
		ID:   p.id,
		Seq:  p.sequence,
		Data: payload,
	}

	msg := &icmp.Message{
		Type: typ,
		Code: 0,
		Body: body,
	}

	msgBytes, err := msg.Marshal(nil)
	if err != nil {
		return err
	}

	for {
		if _, err := conn.WriteTo(msgBytes, dst); err != nil {
			if neterr, ok := err.(*net.OpError); ok {
				if neterr.Err == syscall.ENOBUFS {
					continue
				}
			}
		}
		timer := in_flight{
			Result: Result{
				TS:   now,
				Host: p.Host,
				IP:   addr_string,
				Rtt:  -1 * time.Hour,
			},
		}
		timer.timer = time.AfterFunc(p.Timeout, timer.doTimeout)
		timer.pinger = p
		p.pending[p.sequence] = timer
		p.sequence++
		break
	}

	return nil
}

func (t *in_flight) doTimeout() {
	t.pinger.results <- &t.Result
}

func (p *Pinger) processPacket(recv *packet, results chan *Result) error {
	receivedAt := time.Now()
	var proto int
	if recv.ipv4 {
		proto = protocolICMP
	} else {
		proto = protocolIPv6ICMP
	}

	var m *icmp.Message
	var err error
	if m, err = icmp.ParseMessage(proto, recv.bytes); err != nil {
		return fmt.Errorf("error parsing icmp message: %s", err.Error())
	}

	if m.Type != ipv4.ICMPTypeEchoReply && m.Type != ipv6.ICMPTypeEchoReply {
		// Not an echo reply, ignore it
		return nil
	}

	r := &Result{}

	switch pkt := m.Body.(type) {
	case *icmp.Echo:
		// If we are privileged, we can match icmp.ID
		if p.network == "ip" {
			// Check if reply from same ID
			if pkt.ID != p.id {
				return nil
			}
		}

		// XXX should lock p.pending
		timeout, ok := p.pending[pkt.Seq]
		if ok {
			delete(p.pending, pkt.Seq)
			timeout.timer.Stop()
		}

		if len(pkt.Data) < szLen+tsLen {
			return fmt.Errorf("insufficient data received; got: %d %v",
				len(pkt.Data), pkt.Data)
		}

		size := bytesToInt(pkt.Data[0:szLen])
		if int64(len(pkt.Data)) < szLen+size {
			return fmt.Errorf("payload size mismatch; got: %d, expected %d %v",
				len(pkt.Data), size, pkt.Data)
		}
		offset := int64(szLen)
		r.TS = bytesToTime(pkt.Data[offset : offset+tsLen])
		r.Rtt = receivedAt.Sub(r.TS)
		offset += tsLen
		hostLen := bytesToInt(pkt.Data[offset : offset+szLen])
		if hostLen <= 0 || offset+szLen+hostLen > int64(len(pkt.Data)) {
			return fmt.Errorf("invalid hostlen: %d %v", hostLen, pkt.Data)
		}
		offset += szLen
		r.Host = string(pkt.Data[offset : offset+hostLen])
		if r.Host != p.Host {
			return fmt.Errorf("hostname mismatch, expected %s, got %s %v",
				p.Host, r.Host, pkt.Data)
		}
		offset += hostLen
		addrLen := bytesToInt(pkt.Data[offset : offset+szLen])
		if offset+szLen+addrLen > int64(len(pkt.Data)) {
			return fmt.Errorf("invalid addr len: %d %v", addrLen, pkt.Data)
		}
		offset += szLen
		r.IP = string(pkt.Data[offset : offset+addrLen])

	default:
		// Very bad, not sure how this can happen
		return fmt.Errorf(
			"invalid ICMP echo reply; type: '%T', '%v'", pkt, pkt)
	}

	results <- r

	return nil
}

func (p *Pinger) listen(netProto string) *icmp.PacketConn {
	conn, err := icmp.ListenPacket(netProto, "")
	if err != nil {
		fmt.Printf("Error listening for ICMP packets: %s\n", err.Error())
		close(p.done)
		return nil
	}
	return conn
}

func bytesToTime(b []byte) time.Time {
	var nsec int64
	for i := uint8(0); i < 8; i++ {
		nsec += int64(b[i]) << ((7 - i) * 8)
	}
	return time.Unix(nsec/1000000000, nsec%1000000000)
}

func isIPv4(ip net.IP) bool {
	return len(ip.To4()) == net.IPv4len
}

func isIPv6(ip net.IP) bool {
	return len(ip) == net.IPv6len
}

func timeToBytes(t time.Time) []byte {
	nsec := t.UnixNano()
	b := make([]byte, 8)
	for i := uint8(0); i < 8; i++ {
		b[i] = byte((nsec >> ((7 - i) * 8)) & 0xff)
	}
	return b
}

func bytesToInt(b []byte) int64 {
	return int64(binary.BigEndian.Uint64(b))
}

func intToBytes(tracker int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(tracker))
	return b
}

func start_pinger(db *sql.DB, interval time.Duration) {
	results := ResultWorker(db)
	pingers = make(map[string]*Pinger, 0)
	hosts := get_dests(db)
	for _, host := range hosts {
		pinger, err := NewPinger(host, interval, results)
		if err != nil {
			log.Fatal("could not start pinger for", host, err)
		}
		pinger.SetPrivileged(*privileged)
		pingers[host] = pinger
		go pinger.Run()
	}
}

func stop_pingers() {
	for host, pinger := range pingers {
		pinger.Stop()
		if *verbose {
			log.Println("stopped pinger for", host)
		}
	}
}
