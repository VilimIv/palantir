package main

import (
	"bufio"
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
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/flynn/noise"
	"github.com/gorilla/websocket"
	"golang.zx2c4.com/wireguard/tun"
)

var serverURL = "http://20.250.145.46:8080"
var wsURL = "ws://20.250.145.46:8080"

// UDP framing: prvi byte razlikuje tip poruke
const (
	msgHandshake     byte = 0x01
	msgData          byte = 0x02
	msgProbe         byte = 0x03
	msgProbeResp     byte = 0x04
	msgKeepalive     byte = 0x05
	msgRelayRegister byte = 0x06 // registracija na relay server
	msgRelayData     byte = 0x07 // data proslijeđena kroz relay
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
	UseRelay          bool // true ako probe timeout → koristi relay
	Handshake         *noise.HandshakeState
	HandshakeResponse []byte
	MyRandom          []byte
	SendAEAD          cipher.AEAD
	RecvAEAD          cipher.AEAD
	SendNonce         uint64
	Ready             bool
	mu                sync.Mutex
}

var (
	myVirtualIP string
	myUsername  string

	peers      = map[string]*PeerInfo{}
	peersByUDP = map[string]*PeerInfo{}
	peersMu    sync.RWMutex

	udpConn     net.PacketConn
	tunDev      tun.Device
	relayAddr   *net.UDPAddr // server:8081
	cipherSuite = noise.NewCipherSuite(noise.DH25519, noise.CipherChaChaPoly, noise.HashBLAKE2b)
)

// --- Kriptografija ---

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
		return nil, fmt.Errorf("paket prekratak: %d bytea", len(payload))
	}
	counter := binary.BigEndian.Uint64(payload[0:8])
	nonce := make([]byte, 12)
	binary.BigEndian.PutUint64(nonce[4:], counter)
	return aead.Open(nil, nonce, payload[8:], nil)
}

// --- Relay helper ---

// sendToPeer šalje podatke peeru — direktno ili kroz relay, ovisno o UseRelay
func sendToPeer(peer *PeerInfo, msgType byte, data []byte) {
	if peer.UseRelay && relayAddr != nil {
		// Relay format: 0x07 + targetVIP\0 + originalMsgType + originalPayload
		buf := []byte{msgRelayData}
		buf = append(buf, []byte(peer.VirtualIP)...)
		buf = append(buf, 0) // null terminator
		buf = append(buf, msgType)
		buf = append(buf, data...)
		udpConn.WriteTo(buf, relayAddr)
	} else {
		sendUDP(peer.UDPAddr, msgType, data)
	}
}

// registerRelay šalje registraciju na relay server
func registerRelay() {
	if relayAddr == nil {
		return
	}
	buf := []byte{msgRelayRegister}
	buf = append(buf, []byte(myVirtualIP)...)
	buf = append(buf, 0)
	udpConn.WriteTo(buf, relayAddr)
}

// --- STUN klijent ---

func querySTUN(conn net.PacketConn) *net.UDPAddr {
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
	conn.WriteTo(req, stunServer)

	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 1024)
	n, _, err := conn.ReadFrom(buf)
	conn.SetReadDeadline(time.Time{})
	if err != nil {
		log.Println("STUN timeout:", err)
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
		} else if attrType == 0x0001 && attrLen >= 8 && buf[pos+1] == 0x01 {
			port := binary.BigEndian.Uint16(buf[pos+2 : pos+4])
			ip := make(net.IP, 4)
			copy(ip, buf[pos+4:pos+8])
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

// --- Pomoćne funkcije ---

func prompt(reader *bufio.Reader, label string) string {
	fmt.Print(label)
	text, _ := reader.ReadString('\n')
	return strings.TrimSpace(text)
}

func apiPost(u string, token string, body any) (*http.Response, error) {
	var bodyReader *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}
	req, _ := http.NewRequest("POST", u, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return http.DefaultClient.Do(req)
}

func amInitiator(peerVIP string) bool {
	return myVirtualIP < peerVIP
}

func sendUDP(addr *net.UDPAddr, msgType byte, data []byte) {
	buf := make([]byte, 1+len(data))
	buf[0] = msgType
	copy(buf[1:], data)
	udpConn.WriteTo(buf, addr)
}

// --- Upravljanje peerovima ---

func addPeer(username, virtualIP string, candidates []Candidate) {
	if virtualIP == myVirtualIP {
		return
	}
	peersMu.Lock()
	if _, exists := peers[virtualIP]; exists {
		peersMu.Unlock()
		return
	}
	peer := &PeerInfo{
		Username:   username,
		VirtualIP:  virtualIP,
		Candidates: candidates,
	}
	peers[virtualIP] = peer
	peersMu.Unlock()

	log.Printf("Novi peer: %s (%s), %d kandidata\n", username, virtualIP, len(candidates))
	go probePeer(peer)
}

func removePeer(username string) {
	peersMu.Lock()
	defer peersMu.Unlock()
	for vip, peer := range peers {
		if peer.Username == username {
			if peer.UDPAddr != nil {
				delete(peersByUDP, peer.UDPAddr.String())
			}
			delete(peers, vip)
			log.Printf("Peer uklonjen: %s (%s)\n", username, vip)
			return
		}
	}
}

// --- Probing ---

func probePeer(peer *PeerInfo) {
	var addrs []*net.UDPAddr
	for _, c := range peer.Candidates {
		addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", c.Host, c.Port))
		if err != nil {
			continue
		}
		addrs = append(addrs, addr)
	}
	if len(addrs) == 0 {
		log.Printf("Nema valjanih kandidata za %s\n", peer.Username)
		return
	}

	probe := []byte(myVirtualIP)
	for i := 0; i < 20; i++ {
		peer.mu.Lock()
		resolved := peer.Resolved
		peer.mu.Unlock()
		if resolved {
			return
		}
		for _, addr := range addrs {
			sendUDP(addr, msgProbe, probe)
		}
		if i == 0 {
			log.Printf("Probing %s (%s): %d kandidata...\n", peer.Username, peer.VirtualIP, len(addrs))
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Probe timeout — prelazi na relay
	log.Printf("Probe timeout za %s (%s) — prelazim na relay\n", peer.Username, peer.VirtualIP)
	peer.mu.Lock()
	peer.UseRelay = true
	peer.Resolved = true // sprečava ponovni probe
	peer.mu.Unlock()
	startHandshake(peer)
}

func keepalive() {
	for {
		time.Sleep(15 * time.Second)

		// Re-registriraj na relay
		registerRelay()

		peersMu.RLock()
		for _, peer := range peers {
			peer.mu.Lock()
			if peer.Resolved && !peer.UseRelay && peer.UDPAddr != nil {
				sendUDP(peer.UDPAddr, msgKeepalive, nil)
			}
			peer.mu.Unlock()
		}
		peersMu.RUnlock()
	}
}

func handleProbe(from *net.UDPAddr, msgType byte, payload []byte) {
	senderVIP := string(payload)
	peersMu.RLock()
	peer := peers[senderVIP]
	peersMu.RUnlock()
	if peer == nil {
		return
	}
	if msgType == msgProbe {
		sendUDP(from, msgProbeResp, []byte(myVirtualIP))
	}
	peer.mu.Lock()
	if peer.Resolved {
		peer.mu.Unlock()
		return
	}
	peer.UDPAddr = from
	peer.Resolved = true
	peer.mu.Unlock()

	peersMu.Lock()
	peersByUDP[from.String()] = peer
	peersMu.Unlock()

	log.Printf("Peer %s (%s) dostupan na %s\n", peer.Username, peer.VirtualIP, from)
	startHandshake(peer)
}

// --- Relay data handling ---

// handleRelayData procesira paket primljen kroz relay server
func handleRelayData(payload []byte) {
	// payload = senderVIP\0 + originalMsgType(1B) + originalPayload
	nullIdx := bytes.IndexByte(payload, 0)
	if nullIdx < 0 || nullIdx+2 > len(payload) {
		return
	}
	senderVIP := string(payload[:nullIdx])
	innerMsgType := payload[nullIdx+1]
	innerPayload := payload[nullIdx+2:]

	peersMu.RLock()
	peer := peers[senderVIP]
	peersMu.RUnlock()
	if peer == nil {
		return
	}

	switch innerMsgType {
	case msgHandshake:
		handleHandshakeMsg(peer, innerPayload)

	case msgData:
		peer.mu.Lock()
		if peer.Ready {
			decrypted, err := decryptPacket(peer.RecvAEAD, innerPayload)
			peer.mu.Unlock()
			if err != nil {
				log.Printf("Relay decrypt error od %s: %v\n", peer.Username, err)
				return
			}
			paddedBuf := make([]byte, tunOffset+len(decrypted))
			copy(paddedBuf[tunOffset:], decrypted)
			tunDev.Write([][]byte{paddedBuf}, tunOffset)
		} else {
			peer.mu.Unlock()
		}
	}
}

// --- Noise handshake ---

func startHandshake(peer *PeerInfo) {
	initiator := amInitiator(peer.VirtualIP)

	hs, err := noise.NewHandshakeState(noise.Config{
		CipherSuite: cipherSuite,
		Pattern:     noise.HandshakeNN,
		Initiator:   initiator,
	})
	if err != nil {
		log.Printf("Handshake init error s %s: %v\n", peer.Username, err)
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
			log.Printf("Handshake write error s %s: %v\n", peer.Username, err)
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
				sendToPeer(peer, msgHandshake, msg)
				if i == 0 {
					mode := "direktno"
					if peer.UseRelay {
						mode = "relay"
					}
					log.Printf("Handshake poslan prema %s (%s) [%s]\n", peer.Username, peer.VirtualIP, mode)
				}
				time.Sleep(time.Second)
			}
			log.Printf("Handshake timeout s %s (%s)\n", peer.Username, peer.VirtualIP)
		}()
	} else {
		log.Printf("Čekam handshake od %s (%s)\n", peer.Username, peer.VirtualIP)
	}
}

func handleHandshakeMsg(peer *PeerInfo, data []byte) {
	peer.mu.Lock()
	defer peer.mu.Unlock()

	if peer.Ready {
		if peer.HandshakeResponse != nil {
			sendToPeer(peer, msgHandshake, peer.HandshakeResponse)
		}
		return
	}
	if peer.Handshake == nil {
		return
	}

	peerRandom, cs1, cs2, err := peer.Handshake.ReadMessage(nil, data)
	if err != nil {
		log.Printf("Handshake read error od %s: %v\n", peer.Username, err)
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
		log.Printf("Enkripcija uspostavljena s %s (%s) [AES-256-GCM, %s]\n", peer.Username, peer.VirtualIP, mode)
		return
	}

	myRandom := make([]byte, 32)
	rand.Read(myRandom)
	msg, cs1, cs2, err := peer.Handshake.WriteMessage(nil, myRandom)
	if err != nil {
		return
	}
	peer.HandshakeResponse = msg
	sendToPeer(peer, msgHandshake, msg)

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
		log.Printf("Enkripcija uspostavljena s %s (%s) [AES-256-GCM, %s]\n", peer.Username, peer.VirtualIP, mode)
	}
}

// --- Routing paketa ---

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

func tunToUDP() {
	bufs := make([][]byte, 1)
	bufs[0] = make([]byte, tunOffset+1500)
	sizes := make([]int, 1)

	for {
		n, err := tunDev.Read(bufs, sizes, tunOffset)
		if err != nil {
			continue
		}
		for i := 0; i < n; i++ {
			packet := bufs[i][tunOffset : tunOffset+sizes[i]]
			destIP := getDestIP(packet)
			if destIP == nil || (destIP[0] >= 224 && destIP[0] <= 239) {
				continue
			}

			peersMu.RLock()

			if isBroadcast(destIP) {
				for _, peer := range peers {
					peer.mu.Lock()
					if peer.Ready {
						peer.SendNonce++
						encrypted := encryptPacket(peer.SendAEAD, peer.SendNonce, packet)
						sendToPeer(peer, msgData, encrypted)
					}
					peer.mu.Unlock()
				}
			} else {
				if peer, ok := peers[destIP.String()]; ok {
					peer.mu.Lock()
					if peer.Ready {
						peer.SendNonce++
						encrypted := encryptPacket(peer.SendAEAD, peer.SendNonce, packet)
						sendToPeer(peer, msgData, encrypted)
					}
					peer.mu.Unlock()
				}
			}

			peersMu.RUnlock()
		}
	}
}

func udpToTUN() {
	buf := make([]byte, 2000)

	for {
		n, rawAddr, err := udpConn.ReadFrom(buf)
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

		// Relay data — procesira se posebno (dolazi sa serverove adrese)
		if msgType == msgRelayData {
			handleRelayData(payload)
			continue
		}

		if msgType == msgProbe || msgType == msgProbeResp {
			handleProbe(udpAddr, msgType, payload)
			continue
		}
		if msgType == msgKeepalive {
			continue
		}

		peersMu.RLock()
		peer := peersByUDP[udpAddr.String()]
		peersMu.RUnlock()

		if peer == nil {
			continue
		}

		switch msgType {
		case msgHandshake:
			handleHandshakeMsg(peer, payload)

		case msgData:
			peer.mu.Lock()
			if peer.Ready {
				decrypted, err := decryptPacket(peer.RecvAEAD, payload)
				peer.mu.Unlock()
				if err != nil {
					log.Printf("Decrypt error od %s: %v\n", peer.Username, err)
					continue
				}
				paddedBuf := make([]byte, tunOffset+len(decrypted))
				copy(paddedBuf[tunOffset:], decrypted)
				tunDev.Write([][]byte{paddedBuf}, tunOffset)
			} else {
				peer.mu.Unlock()
			}
		}
	}
}

// --- WebSocket listener ---

func listenWebSocket(wsConn *websocket.Conn) {
	for {
		var raw struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}
		if err := wsConn.ReadJSON(&raw); err != nil {
			log.Println("WebSocket disconnected:", err)
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
			addPeer(data.Username, data.VirtualIP, data.Candidates)
		case "peer_joined":
			var data struct{ Username string `json:"username"` }
			json.Unmarshal(raw.Data, &data)
			log.Printf("Peer %s se pridružio mreži\n", data.Username)
		case "peer_left":
			var data struct{ Username string `json:"username"` }
			json.Unmarshal(raw.Data, &data)
			removePeer(data.Username)
		case "peers_list":
			log.Println("Primljena lista peerova")
		}
	}
}

// --- Main ---

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("=============================")
	fmt.Println("       VLAN Klijent")
	fmt.Println("=============================")

	// 1. Register ili Login
	choice := prompt(reader, "(1) Login  (2) Register: ")
	username := prompt(reader, "Username: ")
	password := prompt(reader, "Password: ")

	if choice == "2" {
		resp, err := apiPost(serverURL+"/register", "", map[string]string{
			"username": username, "password": password,
		})
		if err != nil || resp.StatusCode != 201 {
			log.Fatal("Registracija neuspješna")
		}
		fmt.Println("Registracija uspješna!")
	}

	resp, err := apiPost(serverURL+"/login", "", map[string]string{
		"username": username, "password": password,
	})
	if err != nil || resp.StatusCode != 200 {
		log.Fatal("Login neuspješan")
	}
	var loginResp map[string]string
	json.NewDecoder(resp.Body).Decode(&loginResp)
	token := loginResp["token"]
	myUsername = username
	fmt.Println("Prijavljen!")

	// 2. Kreiraj ili pridruži se mreži
	choice = prompt(reader, "(1) Kreiraj mrežu  (2) Pridruži se: ")
	var networkID string

	if choice == "1" {
		resp, err := apiPost(serverURL+"/network/create", token, nil)
		if err != nil || resp.StatusCode != 201 {
			log.Fatal("Greška pri kreiranju mreže")
		}
		var createResp map[string]string
		json.NewDecoder(resp.Body).Decode(&createResp)
		networkID = createResp["networkID"]
		myVirtualIP = createResp["virtualIP"]
		fmt.Printf("Mreža kreirana! Kod: %s\n", networkID)
		fmt.Println("Podijeli ovaj kod s prijateljem.")
	} else {
		networkID = prompt(reader, "Upiši kod mreže: ")
		resp, err := apiPost(serverURL+"/network/join", token, map[string]string{"networkID": networkID})
		if err != nil || resp.StatusCode != 200 {
			log.Fatal("Greška pri pridruživanju mreži")
		}
		var joinResp struct {
			VirtualIP string `json:"virtualIP"`
			Peers     []struct {
				Username  string `json:"username"`
				VirtualIP string `json:"virtualIP"`
			} `json:"peers"`
		}
		json.NewDecoder(resp.Body).Decode(&joinResp)
		myVirtualIP = joinResp.VirtualIP
		fmt.Printf("Pridružen mreži %s!\n", networkID)
		for _, p := range joinResp.Peers {
			fmt.Printf("  Peer: %s → %s\n", p.Username, p.VirtualIP)
		}
	}
	fmt.Printf("Tvoja virtualna IP: %s\n", myVirtualIP)

	// 3. Kreiraj TUN adapter
	var tunErr error
	tunDev, tunErr = tun.CreateTUN("vlan0", 1500)
	if tunErr != nil {
		log.Fatal("TUN error:", tunErr)
	}
	name, _ := tunDev.Name()
	log.Println("TUN adapter kreiran:", name)

	if runtime.GOOS == "windows" {
		cmd := exec.Command("netsh", "interface", "ip", "set", "address",
			"name=vlan0", "source=static", "addr="+myVirtualIP, "mask=255.255.255.0")
		if out, err := cmd.CombinedOutput(); err != nil {
			log.Fatal("Netsh error:", string(out), err)
		}
	} else {
		cmd := exec.Command("ip", "addr", "add", myVirtualIP+"/24", "dev", "vlan0")
		if out, err := cmd.CombinedOutput(); err != nil {
			log.Fatal("ip addr error:", string(out), err)
		}
		cmd = exec.Command("ip", "link", "set", "vlan0", "up")
		if out, err := cmd.CombinedOutput(); err != nil {
			log.Fatal("ip link error:", string(out), err)
		}
	}
	log.Println("IP adresa postavljena:", myVirtualIP)

	// 4. Otvori UDP socket
	udpConn, err = net.ListenPacket("udp4", ":0")
	if err != nil {
		log.Fatal("UDP error:", err)
	}
	localPort := udpConn.LocalAddr().(*net.UDPAddr).Port
	log.Printf("UDP tunel na portu %d\n", localPort)

	// 5. Postavi relay adresu (isti server, port 8081)
	parsed, _ := url.Parse(serverURL)
	serverHost := parsed.Hostname()
	relayAddr, _ = net.ResolveUDPAddr("udp4", serverHost+":8081")
	log.Printf("Relay server: %s\n", relayAddr)

	// 6. Registriraj se na relay (za slučaj da peer treba relay)
	registerRelay()

	// 7. Otkrij mrežne adrese (lokalna + STUN)
	localIP := getLocalIP()
	log.Printf("Lokalna IP: %s\n", localIP)

	stunAddr := querySTUN(udpConn)
	if stunAddr != nil {
		log.Printf("STUN adresa: %s\n", stunAddr)
	} else {
		log.Println("STUN nije uspio — samo lokalni kandidat")
	}

	var candidates []Candidate
	if localIP != "" {
		candidates = append(candidates, Candidate{Host: localIP, Port: localPort, Type: "local"})
	}
	if stunAddr != nil {
		if stunAddr.IP.String() != localIP || stunAddr.Port != localPort {
			candidates = append(candidates, Candidate{Host: stunAddr.IP.String(), Port: stunAddr.Port, Type: "stun"})
		}
	}
	if len(candidates) == 0 {
		candidates = append(candidates, Candidate{Host: "127.0.0.1", Port: localPort, Type: "local"})
	}
	for _, c := range candidates {
		log.Printf("Kandidat: %s:%d (%s)\n", c.Host, c.Port, c.Type)
	}

	// 8. Spoji se na WebSocket
	wsConn, _, err := websocket.DefaultDialer.Dial(
		fmt.Sprintf("%s/ws?token=%s&networkID=%s", wsURL, token, networkID), nil)
	if err != nil {
		log.Fatal("WebSocket error:", err)
	}
	log.Println("WebSocket spojen!")

	// 9. Objavi kandidate
	wsConn.WriteJSON(map[string]any{
		"type": "announce",
		"data": map[string]any{
			"virtualIP":  myVirtualIP,
			"candidates": candidates,
		},
	})
	log.Println("Announce poslan")

	// 10. Pokreni goroutine
	go listenWebSocket(wsConn)
	go tunToUDP()
	go udpToTUN()
	go keepalive()

	fmt.Println("=============================")
	fmt.Println("  Tunel aktivan! Pritisni Enter za izlaz.")
	fmt.Println("=============================")
	reader.ReadString('\n')
}
