package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/flynn/noise"
	"github.com/gorilla/websocket"
	"golang.zx2c4.com/wireguard/tun"
)

var serverURL = "http://localhost:8080"
var wsURL = "ws://localhost:8080"

// UDP framing: prvi byte razlikuje handshake od podataka
const (
	msgHandshake byte = 0x01
	msgData      byte = 0x02
)

// PeerInfo drži stanje jednog peera: UDP adresu, Noise handshake i cipher stanje
type PeerInfo struct {
	Username          string
	VirtualIP         string
	UDPAddr           *net.UDPAddr
	Handshake         *noise.HandshakeState
	HandshakeResponse []byte // cache za retransmisiju responderovog odgovora
	SendCipher        *noise.CipherState
	RecvCipher        *noise.CipherState
	Ready             bool
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

// addPeer se poziva kad primimo peer_announce — kreira peer i pokreće handshake
func addPeer(username, virtualIP, host string, port int) {
	if virtualIP == myVirtualIP {
		return // to smo mi
	}

	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		log.Printf("Resolve error za %s: %v\n", username, err)
		return
	}

	peersMu.Lock()
	if _, exists := peers[virtualIP]; exists {
		peersMu.Unlock()
		return // već postoji
	}
	peer := &PeerInfo{
		Username:  username,
		VirtualIP: virtualIP,
		UDPAddr:   addr,
	}
	peers[virtualIP] = peer
	peersByUDP[addr.String()] = peer
	peersMu.Unlock()

	log.Printf("Novi peer: %s (%s) na %s\n", username, virtualIP, addr)
	startHandshake(peer)
}

// removePeer briše peer po usernameu (kad primimo peer_left)
func removePeer(username string) {
	peersMu.Lock()
	defer peersMu.Unlock()
	for vip, peer := range peers {
		if peer.Username == username {
			delete(peersByUDP, peer.UDPAddr.String())
			delete(peers, vip)
			log.Printf("Peer uklonjen: %s (%s)\n", username, vip)
			return
		}
	}
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
		// Pripremi prvu poruku pod lockom
		peer.mu.Lock()
		msg, _, _, err := peer.Handshake.WriteMessage(nil, nil)
		peer.mu.Unlock()
		if err != nil {
			log.Printf("Handshake write error s %s: %v\n", peer.Username, err)
			return
		}

		// Retry goroutina — šalje istu poruku dok peer ne odgovori
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

	// Ako je handshake već završen, pošalji cached odgovor (retransmisija za slučaj gubitka)
	if peer.Ready {
		if peer.HandshakeResponse != nil {
			sendUDP(peer.UDPAddr, msgHandshake, peer.HandshakeResponse)
		}
		return
	}

	if peer.Handshake == nil {
		return
	}

	// Pročitaj poruku
	_, cs1, cs2, err := peer.Handshake.ReadMessage(nil, data)
	if err != nil {
		log.Printf("Handshake read error od %s: %v\n", peer.Username, err)
		return
	}

	if cs1 != nil && cs2 != nil {
		// Initiator: handshake završen nakon ReadMessage
		// cs1 = initiator→responder, cs2 = responder→initiator
		peer.SendCipher = cs1
		peer.RecvCipher = cs2
		peer.Ready = true
		peer.Handshake = nil
		log.Printf("Enkripcija uspostavljena s %s (%s)\n", peer.Username, peer.VirtualIP)
		return
	}

	// Responder: treba poslati odgovor
	msg, cs1, cs2, err := peer.Handshake.WriteMessage(nil, nil)
	if err != nil {
		log.Printf("Handshake write error za %s: %v\n", peer.Username, err)
		return
	}

	// Cache odgovor za retransmisiju
	peer.HandshakeResponse = msg
	sendUDP(peer.UDPAddr, msgHandshake, msg)

	if cs1 != nil && cs2 != nil {
		// Responder: handshake završen nakon WriteMessage
		// cs1 = initiator→responder, cs2 = responder→initiator
		peer.RecvCipher = cs1
		peer.SendCipher = cs2
		peer.Ready = true
		peer.Handshake = nil
		log.Printf("Enkripcija uspostavljena s %s (%s)\n", peer.Username, peer.VirtualIP)
	}
}

// --- Routing paketa ---

// getDestIP čita odredišnu IP adresu iz IPv4 headera
func getDestIP(packet []byte) net.IP {
	if len(packet) < 20 {
		return nil
	}
	return net.IP(packet[16:20])
}

// isBroadcast provjerava je li adresa broadcast (255.255.255.255 ili subnet .255)
func isBroadcast(ip net.IP) bool {
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}
	if ip4.Equal(net.IPv4bcast) {
		return true
	}
	// Subnet broadcast za /24 (10.0.0.255)
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

			// Enkriptiraj i pripremi pakete za slanje (pod lockom)
			type outPacket struct {
				addr *net.UDPAddr
				data []byte
			}
			var toSend []outPacket

			peersMu.RLock()

			if isBroadcast(destIP) {
				// Broadcast — pošalji svim peerovima
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
				// Unicast — pronađi peer po odredišnom IP-u
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

			// Šalji izvan locka
			for _, pkt := range toSend {
				sendUDP(pkt.addr, msgData, pkt.data)
			}
		}
	}
}

// udpToTUN čita UDP pakete, identificira peera po izvorišnoj adresi,
// i prosljeđuje dekriptirani paket na TUN ili procesira handshake
func udpToTUN() {
	buf := make([]byte, 2000)

	for {
		n, addr, err := udpConn.ReadFrom(buf)
		if err != nil {
			log.Println("UDP read error:", err)
			continue
		}
		if n < 2 {
			continue
		}

		msgType := buf[0]

		// Kopiraj payload jer se buf reusea za sljedeći read
		payload := make([]byte, n-1)
		copy(payload, buf[1:n])

		// Pronađi peera po UDP adresi
		peersMu.RLock()
		peer := peersByUDP[addr.String()]
		peersMu.RUnlock()

		if peer == nil {
			continue // nepoznat izvor, ignoriraj
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
					continue // neispravan paket
				}
				tunDev.Write([][]byte{decrypted}, 0)
			} else {
				peer.mu.Unlock()
			}
		}
	}
}

// --- WebSocket listener ---

// listenWebSocket sluša signalizacijske poruke i upravlja peerovima
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
				Username  string  `json:"username"`
				VirtualIP string  `json:"virtualIP"`
				Host      string  `json:"host"`
				Port      float64 `json:"port"`
			}
			json.Unmarshal(raw.Data, &data)
			addPeer(data.Username, data.VirtualIP, data.Host, int(data.Port))

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

	cmd := exec.Command("netsh", "interface", "ip", "set", "address",
		"name=vlan0", "source=static", "addr="+myVirtualIP, "mask=255.255.255.0")
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Fatal("Netsh error:", string(out), err)
	}
	log.Println("IP adresa postavljena:", myVirtualIP)

	// 4. Otvori UDP socket
	udpConn, err = net.ListenPacket("udp4", ":0")
	if err != nil {
		log.Fatal("UDP error:", err)
	}
	localPort := udpConn.LocalAddr().(*net.UDPAddr).Port
	log.Printf("UDP tunel na portu %d\n", localPort)

	// 5. Spoji se na WebSocket
	wsConn, _, err := websocket.DefaultDialer.Dial(
		fmt.Sprintf("%s/ws?token=%s&networkID=%s", wsURL, token, networkID), nil)
	if err != nil {
		log.Fatal("WebSocket error:", err)
	}
	log.Println("WebSocket spojen!")

	// 6. Objavi UDP endpoint
	wsConn.WriteJSON(map[string]any{
		"type": "announce",
		"data": map[string]any{
			"virtualIP": myVirtualIP,
			"host":      "127.0.0.1",
			"port":      localPort,
		},
	})
	log.Println("Announce poslan")

	// 7. Pokreni goroutine za networking
	go listenWebSocket(wsConn)
	go tunToUDP()
	go udpToTUN()

	// 8. Čekaj korisnika
	fmt.Println("=============================")
	fmt.Println("  Tunel aktivan! Pritisni Enter za izlaz.")
	fmt.Println("=============================")
	reader.ReadString('\n')
}
