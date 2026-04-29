package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/flynn/noise"
	"github.com/gorilla/websocket"
	"golang.zx2c4.com/wireguard/tun"
)

const (
	msgHandshake     byte = 0x01
	msgData          byte = 0x02
	msgProbe         byte = 0x03
	msgProbeResp     byte = 0x04
	msgKeepalive     byte = 0x05
	msgRelayRegister byte = 0x06
	msgRelayData     byte = 0x07
)

const tunOffset = 16

type Candidate struct {
	Host string `json:"host"`
	Port int    `json:"port"`
	Type string `json:"type"`
}

type PeerInfo struct {
	Username          string
	VirtualIP         string
	Candidates        []Candidate
	UDPAddr           *net.UDPAddr
	Resolved          bool
	UseRelay          bool
	Handshake         *noise.HandshakeState
	HandshakeResponse []byte
	MyRandom          []byte
	SendAEAD          cipher.AEAD
	RecvAEAD          cipher.AEAD
	SendNonce         uint64
	Ready             bool
	mu                sync.Mutex
}

// PeerStatus je struktura za prikaz u GUI-u
type PeerStatus struct {
	Username  string `json:"username"`
	VirtualIP string `json:"virtualIP"`
	Mode      string `json:"mode"` // "P2P" ili "RELAY"
	Ready     bool   `json:"ready"`
}

// Tunnel drži svo stanje tunela
type Tunnel struct {
	ServerURL string
	Token     string
	NetworkID string
	VirtualIP string
	Username  string

	peers      map[string]*PeerInfo
	peersByUDP map[string]*PeerInfo
	peersMu    sync.RWMutex

	udpConn     net.PacketConn
	tunDev      tun.Device
	relayAddr   *net.UDPAddr
	wsConn      *websocket.Conn
	cipherSuite noise.CipherSuite

	onLog        func(string)
	onPeerUpdate func([]PeerStatus)
	onStatus     func(string)

	stopChan chan struct{}
	running  bool
}

func NewTunnel(serverURL string, onLog func(string), onPeerUpdate func([]PeerStatus), onStatus func(string)) *Tunnel {
	return &Tunnel{
		ServerURL:    serverURL,
		peers:        make(map[string]*PeerInfo),
		peersByUDP:   make(map[string]*PeerInfo),
		cipherSuite:  noise.NewCipherSuite(noise.DH25519, noise.CipherChaChaPoly, noise.HashBLAKE2b),
		onLog:        onLog,
		onPeerUpdate: onPeerUpdate,
		onStatus:     onStatus,
		stopChan:     make(chan struct{}),
	}
}

func (t *Tunnel) log(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	log.Println(msg)
	if t.onLog != nil {
		t.onLog(msg)
	}
}

func (t *Tunnel) emitPeers() {
	var statuses []PeerStatus
	t.peersMu.RLock()
	for _, p := range t.peers {
		p.mu.Lock()
		mode := "P2P"
		if p.UseRelay {
			mode = "RELAY"
		}
		statuses = append(statuses, PeerStatus{
			Username:  p.Username,
			VirtualIP: p.VirtualIP,
			Mode:      mode,
			Ready:     p.Ready,
		})
		p.mu.Unlock()
	}
	t.peersMu.RUnlock()
	if t.onPeerUpdate != nil {
		t.onPeerUpdate(statuses)
	}
}

// --- API ---

func (t *Tunnel) apiPost(urlPath string, body any) (*http.Response, error) {
	var bodyReader *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}
	req, _ := http.NewRequest("POST", t.ServerURL+urlPath, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if t.Token != "" {
		req.Header.Set("Authorization", "Bearer "+t.Token)
	}
	return http.DefaultClient.Do(req)
}

func (t *Tunnel) Register(username, password string) error {
	resp, err := t.apiPost("/register", map[string]string{"username": username, "password": password})
	if err != nil {
		return fmt.Errorf("mrežna greška: %v", err)
	}
	if resp.StatusCode != 201 {
		return fmt.Errorf("registracija neuspješna (status %d)", resp.StatusCode)
	}
	return nil
}

func (t *Tunnel) Login(username, password string) error {
	resp, err := t.apiPost("/login", map[string]string{"username": username, "password": password})
	if err != nil {
		return fmt.Errorf("mrežna greška: %v", err)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("krivi username ili password")
	}
	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	t.Token = result["token"]
	t.Username = username
	return nil
}

type CreateResult struct {
	NetworkID string `json:"networkID"`
	VirtualIP string `json:"virtualIP"`
}

func (t *Tunnel) CreateNetwork() (*CreateResult, error) {
	resp, err := t.apiPost("/network/create", nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 201 {
		return nil, fmt.Errorf("greška pri kreiranju mreže")
	}
	var result CreateResult
	json.NewDecoder(resp.Body).Decode(&result)
	t.NetworkID = result.NetworkID
	t.VirtualIP = result.VirtualIP
	return &result, nil
}

type JoinResult struct {
	VirtualIP string `json:"virtualIP"`
	Peers     []struct {
		Username  string `json:"username"`
		VirtualIP string `json:"virtualIP"`
	} `json:"peers"`
}

func (t *Tunnel) JoinNetwork(networkID string) (*JoinResult, error) {
	resp, err := t.apiPost("/network/join", map[string]string{"networkID": networkID})
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("mreža ne postoji")
	}
	var result JoinResult
	json.NewDecoder(resp.Body).Decode(&result)
	t.NetworkID = networkID
	t.VirtualIP = result.VirtualIP
	return &result, nil
}

// --- Crypto ---

func deriveKeys(initiatorRandom, responderRandom []byte) (keyI2R, keyR2I [32]byte) {
	combined := append(initiatorRandom, responderRandom...)
	base := sha256.Sum256(combined)
	keyI2R = sha256.Sum256(append(base[:], []byte("i2r")...))
	keyR2I = sha256.Sum256(append(base[:], []byte("r2i")...))
	return
}

func createAEAD(key [32]byte) cipher.AEAD {
	block, _ := aes.NewCipher(key[:])
	aead, _ := cipher.NewGCM(block)
	return aead
}

func encryptPacket(aead cipher.AEAD, counter uint64, plaintext []byte) []byte {
	nonce := make([]byte, 12)
	binary.BigEndian.PutUint64(nonce[4:], counter)
	encrypted := aead.Seal(nil, nonce, plaintext, nil)
	buf := make([]byte, 8+len(encrypted))
	binary.BigEndian.PutUint64(buf[0:8], counter)
	copy(buf[8:], encrypted)
	return buf
}

func decryptPacket(aead cipher.AEAD, payload []byte) ([]byte, error) {
	if len(payload) < 8 {
		return nil, fmt.Errorf("paket prekratak")
	}
	counter := binary.BigEndian.Uint64(payload[0:8])
	nonce := make([]byte, 12)
	binary.BigEndian.PutUint64(nonce[4:], counter)
	return aead.Open(nil, nonce, payload[8:], nil)
}

// --- UDP helpers ---

func (t *Tunnel) sendUDP(addr *net.UDPAddr, msgType byte, data []byte) {
	buf := make([]byte, 1+len(data))
	buf[0] = msgType
	copy(buf[1:], data)
	t.udpConn.WriteTo(buf, addr)
}

func (t *Tunnel) sendToPeer(peer *PeerInfo, msgType byte, data []byte) {
	if peer.UseRelay && t.relayAddr != nil {
		buf := []byte{msgRelayData}
		buf = append(buf, []byte(peer.VirtualIP)...)
		buf = append(buf, 0)
		buf = append(buf, msgType)
		buf = append(buf, data...)
		t.udpConn.WriteTo(buf, t.relayAddr)
	} else {
		t.sendUDP(peer.UDPAddr, msgType, data)
	}
}

func (t *Tunnel) registerRelay() {
	if t.relayAddr == nil {
		return
	}
	buf := []byte{msgRelayRegister}
	buf = append(buf, []byte(t.VirtualIP)...)
	buf = append(buf, 0)
	t.udpConn.WriteTo(buf, t.relayAddr)
}

// --- STUN ---

func (t *Tunnel) querySTUN() *net.UDPAddr {
	stunServer, err := net.ResolveUDPAddr("udp4", "stun.l.google.com:19302")
	if err != nil {
		return nil
	}
	txID := make([]byte, 12)
	rand.Read(txID)
	req := make([]byte, 20)
	binary.BigEndian.PutUint16(req[0:2], 0x0001)
	binary.BigEndian.PutUint16(req[2:4], 0x0000)
	binary.BigEndian.PutUint32(req[4:8], 0x2112A442)
	copy(req[8:20], txID)
	t.udpConn.WriteTo(req, stunServer)

	t.udpConn.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 1024)
	n, _, err := t.udpConn.ReadFrom(buf)
	t.udpConn.SetReadDeadline(time.Time{})
	if err != nil {
		return nil
	}
	if n < 20 || binary.BigEndian.Uint16(buf[0:2]) != 0x0101 {
		return nil
	}

	msgLen := int(binary.BigEndian.Uint16(buf[2:4]))
	pos := 20
	for pos+4 <= 20+msgLen && pos+4 <= n {
		attrType := binary.BigEndian.Uint16(buf[pos : pos+2])
		attrLen := int(binary.BigEndian.Uint16(buf[pos+2 : pos+4]))
		pos += 4
		if pos+attrLen > n {
			break
		}
		if attrType == 0x0020 && attrLen >= 8 && buf[pos+1] == 0x01 {
			port := binary.BigEndian.Uint16(buf[pos+2:pos+4]) ^ 0x2112
			ip := make(net.IP, 4)
			binary.BigEndian.PutUint32(ip, binary.BigEndian.Uint32(buf[pos+4:pos+8])^0x2112A442)
			return &net.UDPAddr{IP: ip, Port: int(port)}
		}
		pos += attrLen
		if attrLen%4 != 0 {
			pos += 4 - (attrLen % 4)
		}
	}
	return nil
}

func getLocalIP() string {
	conn, err := net.Dial("udp4", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}

// --- Peer management ---

func (t *Tunnel) addPeer(username, virtualIP string, candidates []Candidate) {
	if virtualIP == t.VirtualIP {
		return
	}
	t.peersMu.Lock()
	if _, exists := t.peers[virtualIP]; exists {
		t.peersMu.Unlock()
		return
	}
	peer := &PeerInfo{Username: username, VirtualIP: virtualIP, Candidates: candidates}
	t.peers[virtualIP] = peer
	t.peersMu.Unlock()

	t.log("Novi peer: %s (%s), %d kandidata", username, virtualIP, len(candidates))
	t.emitPeers()
	go t.probePeer(peer)
}

func (t *Tunnel) removePeer(username string) {
	t.peersMu.Lock()
	defer t.peersMu.Unlock()
	for vip, peer := range t.peers {
		if peer.Username == username {
			if peer.UDPAddr != nil {
				delete(t.peersByUDP, peer.UDPAddr.String())
			}
			delete(t.peers, vip)
			t.log("Peer uklonjen: %s (%s)", username, vip)
			go t.emitPeers()
			return
		}
	}
}

// --- Probing ---

func (t *Tunnel) probePeer(peer *PeerInfo) {
	var addrs []*net.UDPAddr
	for _, c := range peer.Candidates {
		addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", c.Host, c.Port))
		if err == nil {
			addrs = append(addrs, addr)
		}
	}
	if len(addrs) == 0 {
		return
	}

	probe := []byte(t.VirtualIP)
	for i := 0; i < 20; i++ {
		peer.mu.Lock()
		resolved := peer.Resolved
		peer.mu.Unlock()
		if resolved {
			return
		}
		for _, addr := range addrs {
			t.sendUDP(addr, msgProbe, probe)
		}
		if i == 0 {
			t.log("Probing %s (%s)...", peer.Username, peer.VirtualIP)
		}
		time.Sleep(500 * time.Millisecond)
	}

	t.log("Probe timeout za %s — prelazim na relay", peer.Username)
	peer.mu.Lock()
	peer.UseRelay = true
	peer.Resolved = true
	peer.mu.Unlock()
	t.emitPeers()
	t.startHandshake(peer)
}

func (t *Tunnel) handleProbe(from *net.UDPAddr, msgType byte, payload []byte) {
	senderVIP := string(payload)
	t.peersMu.RLock()
	peer := t.peers[senderVIP]
	t.peersMu.RUnlock()
	if peer == nil {
		return
	}
	if msgType == msgProbe {
		t.sendUDP(from, msgProbeResp, []byte(t.VirtualIP))
	}
	peer.mu.Lock()
	if peer.Resolved {
		peer.mu.Unlock()
		return
	}
	peer.UDPAddr = from
	peer.Resolved = true
	peer.mu.Unlock()

	t.peersMu.Lock()
	t.peersByUDP[from.String()] = peer
	t.peersMu.Unlock()

	t.log("Peer %s dostupan na %s", peer.Username, from)
	t.emitPeers()
	t.startHandshake(peer)
}

func (t *Tunnel) handleRelayData(payload []byte) {
	nullIdx := bytes.IndexByte(payload, 0)
	if nullIdx < 0 || nullIdx+2 > len(payload) {
		return
	}
	senderVIP := string(payload[:nullIdx])
	innerMsgType := payload[nullIdx+1]
	innerPayload := payload[nullIdx+2:]

	t.peersMu.RLock()
	peer := t.peers[senderVIP]
	t.peersMu.RUnlock()
	if peer == nil {
		return
	}

	switch innerMsgType {
	case msgHandshake:
		t.handleHandshakeMsg(peer, innerPayload)
	case msgData:
		peer.mu.Lock()
		if peer.Ready {
			decrypted, err := decryptPacket(peer.RecvAEAD, innerPayload)
			peer.mu.Unlock()
			if err != nil {
				return
			}
			paddedBuf := make([]byte, tunOffset+len(decrypted))
			copy(paddedBuf[tunOffset:], decrypted)
			t.tunDev.Write([][]byte{paddedBuf}, tunOffset)
		} else {
			peer.mu.Unlock()
		}
	}
}

// --- Handshake ---

func (t *Tunnel) amInitiator(peerVIP string) bool {
	return t.VirtualIP < peerVIP
}

func (t *Tunnel) startHandshake(peer *PeerInfo) {
	initiator := t.amInitiator(peer.VirtualIP)
	hs, err := noise.NewHandshakeState(noise.Config{
		CipherSuite: t.cipherSuite,
		Pattern:     noise.HandshakeNN,
		Initiator:   initiator,
	})
	if err != nil {
		return
	}

	peer.mu.Lock()
	peer.Handshake = hs
	peer.mu.Unlock()

	if initiator {
		myRandom := make([]byte, 32)
		rand.Read(myRandom)
		peer.mu.Lock()
		peer.MyRandom = myRandom
		msg, _, _, err := peer.Handshake.WriteMessage(nil, myRandom)
		peer.mu.Unlock()
		if err != nil {
			return
		}

		go func() {
			for i := 0; i < 10; i++ {
				peer.mu.Lock()
				ready := peer.Ready
				peer.mu.Unlock()
				if ready {
					return
				}
				t.sendToPeer(peer, msgHandshake, msg)
				if i == 0 {
					mode := "direktno"
					if peer.UseRelay {
						mode = "relay"
					}
					t.log("Handshake poslan prema %s [%s]", peer.Username, mode)
				}
				time.Sleep(time.Second)
			}
		}()
	}
}

func (t *Tunnel) handleHandshakeMsg(peer *PeerInfo, data []byte) {
	peer.mu.Lock()
	defer peer.mu.Unlock()

	if peer.Ready {
		if peer.HandshakeResponse != nil {
			t.sendToPeer(peer, msgHandshake, peer.HandshakeResponse)
		}
		return
	}
	if peer.Handshake == nil {
		return
	}

	peerRandom, cs1, cs2, err := peer.Handshake.ReadMessage(nil, data)
	if err != nil {
		return
	}

	if cs1 != nil && cs2 != nil {
		keyI2R, keyR2I := deriveKeys(peer.MyRandom, peerRandom)
		peer.SendAEAD = createAEAD(keyI2R)
		peer.RecvAEAD = createAEAD(keyR2I)
		peer.Ready = true
		peer.Handshake = nil
		peer.MyRandom = nil
		mode := "P2P"
		if peer.UseRelay {
			mode = "RELAY"
		}
		t.log("Enkripcija uspostavljena s %s [%s]", peer.Username, mode)
		if t.onStatus != nil {
			t.onStatus("connected")
		}
		go t.emitPeers()
		return
	}

	myRandom := make([]byte, 32)
	rand.Read(myRandom)
	msg, cs1, cs2, err := peer.Handshake.WriteMessage(nil, myRandom)
	if err != nil {
		return
	}
	peer.HandshakeResponse = msg
	t.sendToPeer(peer, msgHandshake, msg)

	if cs1 != nil && cs2 != nil {
		keyI2R, keyR2I := deriveKeys(peerRandom, myRandom)
		peer.SendAEAD = createAEAD(keyR2I)
		peer.RecvAEAD = createAEAD(keyI2R)
		peer.Ready = true
		peer.Handshake = nil
		mode := "P2P"
		if peer.UseRelay {
			mode = "RELAY"
		}
		t.log("Enkripcija uspostavljena s %s [%s]", peer.Username, mode)
		if t.onStatus != nil {
			t.onStatus("connected")
		}
		go t.emitPeers()
	}
}

// --- Routing ---

func getDestIP(packet []byte) net.IP {
	if len(packet) < 20 || packet[0]>>4 != 4 {
		return nil
	}
	return net.IP(packet[16:20])
}

func isBroadcast(ip net.IP) bool {
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}
	return ip4.Equal(net.IPv4bcast) || ip4[3] == 255
}

func (t *Tunnel) tunToUDP() {
	bufs := make([][]byte, 1)
	bufs[0] = make([]byte, tunOffset+1500)
	sizes := make([]int, 1)

	for {
		select {
		case <-t.stopChan:
			return
		default:
		}

		n, err := t.tunDev.Read(bufs, sizes, tunOffset)
		if err != nil {
			continue
		}
		for i := 0; i < n; i++ {
			packet := bufs[i][tunOffset : tunOffset+sizes[i]]
			destIP := getDestIP(packet)
			if destIP == nil || (destIP[0] >= 224 && destIP[0] <= 239) {
				continue
			}

			t.peersMu.RLock()
			if isBroadcast(destIP) {
				for _, peer := range t.peers {
					peer.mu.Lock()
					if peer.Ready {
						peer.SendNonce++
						encrypted := encryptPacket(peer.SendAEAD, peer.SendNonce, packet)
						t.sendToPeer(peer, msgData, encrypted)
					}
					peer.mu.Unlock()
				}
			} else {
				if peer, ok := t.peers[destIP.String()]; ok {
					peer.mu.Lock()
					if peer.Ready {
						peer.SendNonce++
						encrypted := encryptPacket(peer.SendAEAD, peer.SendNonce, packet)
						t.sendToPeer(peer, msgData, encrypted)
					}
					peer.mu.Unlock()
				}
			}
			t.peersMu.RUnlock()
		}
	}
}

func (t *Tunnel) udpToTUN() {
	buf := make([]byte, 2000)
	for {
		select {
		case <-t.stopChan:
			return
		default:
		}

		n, rawAddr, err := t.udpConn.ReadFrom(buf)
		if err != nil {
			continue
		}
		if n < 2 {
			continue
		}

		udpAddr, ok := rawAddr.(*net.UDPAddr)
		if !ok {
			continue
		}

		msgType := buf[0]
		payload := make([]byte, n-1)
		copy(payload, buf[1:n])

		if msgType == msgRelayData {
			t.handleRelayData(payload)
			continue
		}
		if msgType == msgProbe || msgType == msgProbeResp {
			t.handleProbe(udpAddr, msgType, payload)
			continue
		}
		if msgType == msgKeepalive {
			continue
		}

		t.peersMu.RLock()
		peer := t.peersByUDP[udpAddr.String()]
		t.peersMu.RUnlock()
		if peer == nil {
			continue
		}

		switch msgType {
		case msgHandshake:
			t.handleHandshakeMsg(peer, payload)
		case msgData:
			peer.mu.Lock()
			if peer.Ready {
				decrypted, err := decryptPacket(peer.RecvAEAD, payload)
				peer.mu.Unlock()
				if err != nil {
					continue
				}
				paddedBuf := make([]byte, tunOffset+len(decrypted))
				copy(paddedBuf[tunOffset:], decrypted)
				t.tunDev.Write([][]byte{paddedBuf}, tunOffset)
			} else {
				peer.mu.Unlock()
			}
		}
	}
}

func (t *Tunnel) keepalive() {
	for {
		select {
		case <-t.stopChan:
			return
		case <-time.After(15 * time.Second):
		}
		t.registerRelay()
		t.peersMu.RLock()
		for _, peer := range t.peers {
			peer.mu.Lock()
			if peer.Resolved && !peer.UseRelay && peer.UDPAddr != nil {
				t.sendUDP(peer.UDPAddr, msgKeepalive, nil)
			}
			peer.mu.Unlock()
		}
		t.peersMu.RUnlock()
	}
}

func (t *Tunnel) listenWebSocket() {
	for {
		var raw struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}
		if err := t.wsConn.ReadJSON(&raw); err != nil {
			t.log("WebSocket disconnected")
			return
		}
		switch raw.Type {
		case "peer_announce":
			var data struct {
				Username   string      `json:"username"`
				VirtualIP  string      `json:"virtualIP"`
				Candidates []Candidate `json:"candidates"`
			}
			json.Unmarshal(raw.Data, &data)
			t.addPeer(data.Username, data.VirtualIP, data.Candidates)
		case "peer_joined":
			var data struct{ Username string `json:"username"` }
			json.Unmarshal(raw.Data, &data)
			t.log("Peer %s se pridružio mreži", data.Username)
		case "peer_left":
			var data struct{ Username string `json:"username"` }
			json.Unmarshal(raw.Data, &data)
			t.removePeer(data.Username)
		}
	}
}

// --- Start/Stop ---

func (t *Tunnel) Start() error {
	if t.onStatus != nil {
		t.onStatus("connecting")
	}
	t.stopChan = make(chan struct{})

	// TUN
	var err error
	t.tunDev, err = tun.CreateTUN("vlan0", 1500)
	if err != nil {
		return fmt.Errorf("TUN error: %v", err)
	}
	name, _ := t.tunDev.Name()
	t.log("TUN adapter: %s", name)

	if runtime.GOOS == "windows" {
		cmd := exec.Command("netsh", "interface", "ip", "set", "address",
			"name=vlan0", "source=static", "addr="+t.VirtualIP, "mask=255.255.255.0")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("netsh error: %s %v", string(out), err)
		}
	} else {
		cmd := exec.Command("ip", "addr", "add", t.VirtualIP+"/24", "dev", "vlan0")
		cmd.CombinedOutput()
		cmd = exec.Command("ip", "link", "set", "vlan0", "up")
		cmd.CombinedOutput()
	}
	t.log("IP: %s", t.VirtualIP)

	// UDP
	t.udpConn, err = net.ListenPacket("udp4", ":0")
	if err != nil {
		return fmt.Errorf("UDP error: %v", err)
	}
	localPort := t.udpConn.LocalAddr().(*net.UDPAddr).Port
	t.log("UDP port: %d", localPort)

	// Relay
	parsed, _ := url.Parse(t.ServerURL)
	t.relayAddr, _ = net.ResolveUDPAddr("udp4", parsed.Hostname()+":8081")
	t.registerRelay()

	// STUN
	localIP := getLocalIP()
	t.log("Lokalna IP: %s", localIP)
	stunAddr := t.querySTUN()
	if stunAddr != nil {
		t.log("STUN: %s", stunAddr)
	}

	var candidates []Candidate
	if localIP != "" {
		candidates = append(candidates, Candidate{Host: localIP, Port: localPort, Type: "local"})
	}
	if stunAddr != nil && (stunAddr.IP.String() != localIP || stunAddr.Port != localPort) {
		candidates = append(candidates, Candidate{Host: stunAddr.IP.String(), Port: stunAddr.Port, Type: "stun"})
	}
	if len(candidates) == 0 {
		candidates = append(candidates, Candidate{Host: "127.0.0.1", Port: localPort, Type: "local"})
	}

	// WebSocket
	wsURL := strings.Replace(t.ServerURL, "http", "ws", 1)
	t.wsConn, _, err = websocket.DefaultDialer.Dial(
		fmt.Sprintf("%s/ws?token=%s&networkID=%s", wsURL, t.Token, t.NetworkID), nil)
	if err != nil {
		return fmt.Errorf("WebSocket error: %v", err)
	}

	t.wsConn.WriteJSON(map[string]any{
		"type": "announce",
		"data": map[string]any{
			"virtualIP":  t.VirtualIP,
			"candidates": candidates,
		},
	})
	t.log("Spojen na mrežu %s", t.NetworkID)

	// Goroutine
	go t.listenWebSocket()
	go t.tunToUDP()
	go t.udpToTUN()
	go t.keepalive()

	t.running = true
	return nil
}

func (t *Tunnel) Stop() {
	if !t.running {
		return
	}
	close(t.stopChan)
	if t.wsConn != nil {
		t.wsConn.Close()
	}
	if t.udpConn != nil {
		t.udpConn.Close()
	}
	if t.tunDev != nil {
		t.tunDev.Close()
	}
	t.running = false
	t.peers = make(map[string]*PeerInfo)
	t.peersByUDP = make(map[string]*PeerInfo)
	if t.onStatus != nil {
		t.onStatus("disconnected")
	}
	t.log("Tunel zatvoren")
}

func (t *Tunnel) GetPeers() []PeerStatus {
	var statuses []PeerStatus
	t.peersMu.RLock()
	for _, p := range t.peers {
		p.mu.Lock()
		mode := "P2P"
		if p.UseRelay {
			mode = "RELAY"
		}
		statuses = append(statuses, PeerStatus{
			Username:  p.Username,
			VirtualIP: p.VirtualIP,
			Mode:      mode,
			Ready:     p.Ready,
		})
		p.mu.Unlock()
	}
	t.peersMu.RUnlock()
	return statuses
}
