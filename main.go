package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
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
	msgHandshake byte = 0x01
	msgData      byte = 0x02
	msgProbe     byte = 0x03 // probe za otkrivanje putanje do peera
	msgProbeResp byte = 0x04 // odgovor na probe
)

// Candidate predstavlja jedan mrežni endpoint na kojem je peer dostupan
type Candidate struct {
	Host string `json:"host"`
	Port int    `json:"port"`
	Type string `json:"type"` // "local" ili "stun"
}

// PeerInfo drži stanje jednog peera: kandidate, handshake i cipher stanje
type PeerInfo struct {
	Username          string
	VirtualIP         string
	Candidates        []Candidate // lista kandidata za probing
	UDPAddr           *net.UDPAddr
	Resolved          bool // true kad je probing našao radnu putanju
	Handshake         *noise.HandshakeState
	HandshakeResponse []byte // cache za retransmisiju responderovog odgovora
	SendCipher        *noise.CipherState
	RecvCipher        *noise.CipherState
	Ready             bool // true kad je Noise handshake završen
	mu                sync.Mutex
}

var (
	myVirtualIP string
	myUsername  string

	peers      = map[string]*PeerInfo{} // virtualIP → peer
	peersByUDP = map[string]*PeerInfo{} // udpAddr.String() → peer
	peersMu    sync.RWMutex

	udpConn     net.PacketConn
	tunDev      tun.Device
	cipherSuite = noise.NewCipherSuite(noise.DH25519, noise.CipherChaChaPoly, noise.HashBLAKE2b)
)

// --- STUN klijent ---

// querySTUN šalje STUN Binding Request i vraća javnu IP:port adresu.
// Koristi isti UDP socket kao tunel da STUN vrati mapping za taj port.
func querySTUN(conn net.PacketConn) *net.UDPAddr {
	stunServer, err := net.ResolveUDPAddr("udp4", "stun.l.google.com:19302")
	if err != nil {
		log.Println("STUN resolve error:", err)
		return nil
	}

	// STUN Binding Request (RFC 5389)
	txID := make([]byte, 12)
	rand.Read(txID)

	req := make([]byte, 20)
	binary.BigEndian.PutUint16(req[0:2], 0x0001)     // Message Type: Binding Request
	binary.BigEndian.PutUint16(req[2:4], 0x0000)     // Message Length: 0
	binary.BigEndian.PutUint32(req[4:8], 0x2112A442) // Magic Cookie
	copy(req[8:20], txID)

	conn.WriteTo(req, stunServer)

	// Čekaj odgovor s timeoutom
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 1024)
	n, _, err := conn.ReadFrom(buf)
	conn.SetReadDeadline(time.Time{}) // resetiraj deadline
	if err != nil {
		log.Println("STUN timeout ili greška:", err)
		return nil
	}

	// Parsiraj STUN Binding Response
	if n < 20 {
		return nil
	}
	msgType := binary.BigEndian.Uint16(buf[0:2])
	if msgType != 0x0101 { // Binding Success Response
		return nil
	}

	// Iteriraj kroz atribute tražeći XOR-MAPPED-ADDRESS ili MAPPED-ADDRESS
	msgLen := int(binary.BigEndian.Uint16(buf[2:4]))
	pos := 20
	for pos+4 <= 20+msgLen && pos+4 <= n {
		attrType := binary.BigEndian.Uint16(buf[pos : pos+2])
		attrLen := int(binary.BigEndian.Uint16(buf[pos+2 : pos+4]))
		pos += 4

		if pos+attrLen > n {
			break
		}

		if attrType == 0x0020 && attrLen >= 8 { // XOR-MAPPED-ADDRESS
			family := buf[pos+1]
			if family == 0x01 { // IPv4
				port := binary.BigEndian.Uint16(buf[pos+2:pos+4]) ^ 0x2112
				ip := make(net.IP, 4)
				binary.BigEndian.PutUint32(ip, binary.BigEndian.Uint32(buf[pos+4:pos+8])^0x2112A442)
				return &net.UDPAddr{IP: ip, Port: int(port)}
			}
		} else if attrType == 0x0001 && attrLen >= 8 { // MAPPED-ADDRESS (fallback)
			family := buf[pos+1]
			if family == 0x01 {
				port := binary.BigEndian.Uint16(buf[pos+2 : pos+4])
				ip := make(net.IP, 4)
				copy(ip, buf[pos+4:pos+8])
				return &net.UDPAddr{IP: ip, Port: int(port)}
			}
		}

		// Atributi su poravnati na 4 bytea
		pos += attrLen
		if attrLen%4 != 0 {
			pos += 4 - (attrLen % 4)
		}
	}

	return nil
}

// getLocalIP vraća LAN IP adresu ovog stroja
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

func apiPost(url string, token string, body any) (*http.Response, error) {
	var bodyReader *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}
	req, _ := http.NewRequest("POST", url, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return http.DefaultClient.Do(req)
}

// Niži virtualIP je initiator u Noise handshakeu — deterministično određivanje uloga
func amInitiator(peerVIP string) bool {
	return myVirtualIP < peerVIP
}

// sendUDP šalje UDP paket s type prefiksom
func sendUDP(addr *net.UDPAddr, msgType byte, data []byte) {
	buf := make([]byte, 1+len(data))
	buf[0] = msgType
	copy(buf[1:], data)
	udpConn.WriteTo(buf, addr)
}

// --- Upravljanje peerovima ---

// addPeer se poziva kad primimo peer_announce — kreira peer i pokreće probing
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
	// NE registriramo u peersByUDP — to će napraviti handleProbe kad se peer resolva
	peersMu.Unlock()

	log.Printf("Novi peer: %s (%s), %d kandidata\n", username, virtualIP, len(candidates))
	go probePeer(peer)
}

// removePeer briše peer po usernameu (kad primimo peer_left)
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

// probePeer šalje probe pakete na sve kandidate peera dok ne dobije odgovor.
// Ovo istovremeno obavlja UDP hole punching — slanje na STUN adresu kreira NAT mapping.
func probePeer(peer *PeerInfo) {
	// Resolviraj candidate adrese jednom
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

	for i := 0; i < 20; i++ { // 20 × 500ms = 10 sekundi
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
	log.Printf("Probe timeout za %s (%s)\n", peer.Username, peer.VirtualIP)
}

// handleProbe procesira primljeni probe ili probe response.
// Kad prvi probe stigne od peera, resolvira se njegova adresa i kreće handshake.
func handleProbe(from *net.UDPAddr, msgType byte, payload []byte) {
	senderVIP := string(payload)

	peersMu.RLock()
	peer := peers[senderVIP]
	peersMu.RUnlock()

	if peer == nil {
		return // announce za ovog peera još nije stigao
	}

	// Ako je ovo probe (ne response), pošalji response
	if msgType == msgProbe {
		sendUDP(from, msgProbeResp, []byte(myVirtualIP))
	}

	// Resolvaj peera ako još nije resolviran
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

// --- Noise handshake ---

// startHandshake kreira Noise NN handshake za peer.
// Peer s nižim virtualIP-om je initiator i šalje prvi.
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
		peer.mu.Lock()
		msg, _, _, err := peer.Handshake.WriteMessage(nil, nil)
		peer.mu.Unlock()
		if err != nil {
			log.Printf("Handshake write error s %s: %v\n", peer.Username, err)
			return
		}

		// Retry goroutina
		go func() {
			for i := 0; i < 10; i++ {
				peer.mu.Lock()
				ready := peer.Ready
				peer.mu.Unlock()
				if ready {
					return
				}
				sendUDP(peer.UDPAddr, msgHandshake, msg)
				if i == 0 {
					log.Printf("Handshake poslan prema %s (%s)\n", peer.Username, peer.VirtualIP)
				}
				time.Sleep(time.Second)
			}
			log.Printf("Handshake timeout s %s (%s)\n", peer.Username, peer.VirtualIP)
		}()
	} else {
		log.Printf("Čekam handshake od %s (%s)\n", peer.Username, peer.VirtualIP)
	}
}

// handleHandshakeMsg procesira primljenu handshake poruku od peera
func handleHandshakeMsg(peer *PeerInfo, data []byte) {
	peer.mu.Lock()
	defer peer.mu.Unlock()

	if peer.Ready {
		if peer.HandshakeResponse != nil {
			sendUDP(peer.UDPAddr, msgHandshake, peer.HandshakeResponse)
		}
		return
	}

	if peer.Handshake == nil {
		return
	}

	_, cs1, cs2, err := peer.Handshake.ReadMessage(nil, data)
	if err != nil {
		log.Printf("Handshake read error od %s: %v\n", peer.Username, err)
		return
	}

	if cs1 != nil && cs2 != nil {
		// Initiator: handshake završen
		peer.SendCipher = cs1
		peer.RecvCipher = cs2
		peer.Ready = true
		peer.Handshake = nil
		log.Printf("Enkripcija uspostavljena s %s (%s)\n", peer.Username, peer.VirtualIP)
		return
	}

	// Responder: šalje odgovor
	msg, cs1, cs2, err := peer.Handshake.WriteMessage(nil, nil)
	if err != nil {
		log.Printf("Handshake write error za %s: %v\n", peer.Username, err)
		return
	}

	peer.HandshakeResponse = msg
	sendUDP(peer.UDPAddr, msgHandshake, msg)

	if cs1 != nil && cs2 != nil {
		peer.RecvCipher = cs1
		peer.SendCipher = cs2
		peer.Ready = true
		peer.Handshake = nil
		log.Printf("Enkripcija uspostavljena s %s (%s)\n", peer.Username, peer.VirtualIP)
	}
}

// --- Routing paketa ---

func getDestIP(packet []byte) net.IP {
	if len(packet) < 20 {
		return nil
	}
	return net.IP(packet[16:20])
}

func isBroadcast(ip net.IP) bool {
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}
	if ip4.Equal(net.IPv4bcast) {
		return true
	}
	if ip4[3] == 255 {
		return true
	}
	return false
}

// tunToUDP čita pakete s TUN adaptera i šalje ih pravom peeru (ili svima za broadcast)
func tunToUDP() {
	bufs := make([][]byte, 1)
	bufs[0] = make([]byte, 1500)
	sizes := make([]int, 1)

	for {
		n, err := tunDev.Read(bufs, sizes, 0)
		if err != nil {
			log.Println("TUN read error:", err)
			continue
		}

		for i := 0; i < n; i++ {
			packet := bufs[i][:sizes[i]]
			destIP := getDestIP(packet)
			if destIP == nil {
				continue
			}

			type outPacket struct {
				addr *net.UDPAddr
				data []byte
			}
			var toSend []outPacket

			peersMu.RLock()

			if isBroadcast(destIP) {
				for _, peer := range peers {
					peer.mu.Lock()
					if peer.Ready {
						encrypted, err := peer.SendCipher.Encrypt(nil, nil, packet)
						if err == nil {
							toSend = append(toSend, outPacket{peer.UDPAddr, encrypted})
						}
					}
					peer.mu.Unlock()
				}
			} else {
				if peer, ok := peers[destIP.String()]; ok {
					peer.mu.Lock()
					if peer.Ready {
						encrypted, err := peer.SendCipher.Encrypt(nil, nil, packet)
						if err == nil {
							toSend = append(toSend, outPacket{peer.UDPAddr, encrypted})
						}
					}
					peer.mu.Unlock()
				}
			}

			peersMu.RUnlock()

			for _, pkt := range toSend {
				sendUDP(pkt.addr, msgData, pkt.data)
			}
		}
	}
}

// udpToTUN čita UDP pakete — procesira probe, handshake ili enkriptirane podatke
func udpToTUN() {
	buf := make([]byte, 2000)

	for {
		n, rawAddr, err := udpConn.ReadFrom(buf)
		if err != nil {
			log.Println("UDP read error:", err)
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

		// Probe paketi se procesiraju PRIJE peersByUDP lookupa jer
		// neresolvirani peeri još nisu u toj mapi
		if msgType == msgProbe || msgType == msgProbeResp {
			handleProbe(udpAddr, msgType, payload)
			continue
		}

		// Handshake i data — lookup po UDP adresi
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
				decrypted, err := peer.RecvCipher.Decrypt(nil, nil, payload)
				peer.mu.Unlock()
				if err != nil {
					continue
				}
				tunDev.Write([][]byte{decrypted}, 0)
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
			var data struct {
				Username string `json:"username"`
			}
			json.Unmarshal(raw.Data, &data)
			log.Printf("Peer %s se pridružio mreži\n", data.Username)

		case "peer_left":
			var data struct {
				Username string `json:"username"`
			}
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
		resp, err := apiPost(serverURL+"/network/join", token, map[string]string{
			"networkID": networkID,
		})
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
		// Linux: ip addr add + ip link set up
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

	// 5. Otkrij mrežne adrese (lokalna + STUN)
	localIP := getLocalIP()
	log.Printf("Lokalna IP: %s\n", localIP)

	stunAddr := querySTUN(udpConn)
	if stunAddr != nil {
		log.Printf("STUN adresa: %s\n", stunAddr)
	} else {
		log.Println("STUN nije uspio — samo lokalni kandidat")
	}

	// Gradi listu kandidata
	var candidates []Candidate
	if localIP != "" {
		candidates = append(candidates, Candidate{
			Host: localIP,
			Port: localPort,
			Type: "local",
		})
	}
	if stunAddr != nil {
		// Deduplikacija: ne dodaj ako je isto kao lokalni
		if stunAddr.IP.String() != localIP || stunAddr.Port != localPort {
			candidates = append(candidates, Candidate{
				Host: stunAddr.IP.String(),
				Port: stunAddr.Port,
				Type: "stun",
			})
		}
	}
	// Fallback za development
	if len(candidates) == 0 {
		candidates = append(candidates, Candidate{
			Host: "127.0.0.1",
			Port: localPort,
			Type: "local",
		})
		log.Println("UPOZORENJE: koristi se localhost (STUN i LAN discovery neuspješni)")
	}

	for _, c := range candidates {
		log.Printf("Kandidat: %s:%d (%s)\n", c.Host, c.Port, c.Type)
	}

	// 6. Spoji se na WebSocket
	wsConn, _, err := websocket.DefaultDialer.Dial(
		fmt.Sprintf("%s/ws?token=%s&networkID=%s", wsURL, token, networkID), nil)
	if err != nil {
		log.Fatal("WebSocket error:", err)
	}
	log.Println("WebSocket spojen!")

	// 7. Objavi kandidate
	wsConn.WriteJSON(map[string]any{
		"type": "announce",
		"data": map[string]any{
			"virtualIP":  myVirtualIP,
			"candidates": candidates,
		},
	})
	log.Println("Announce poslan")

	// 8. Pokreni goroutine za networking
	go listenWebSocket(wsConn)
	go tunToUDP()
	go udpToTUN()

	// 9. Čekaj korisnika
	fmt.Println("=============================")
	fmt.Println("  Tunel aktivan! Pritisni Enter za izlaz.")
	fmt.Println("=============================")
	reader.ReadString('\n')
}
