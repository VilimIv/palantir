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
	"time"

	"github.com/flynn/noise"
	"github.com/gorilla/websocket"
	"golang.zx2c4.com/wireguard/tun"
)

var serverURL = "http://localhost:8080"
var wsURL = "ws://localhost:8080"

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
	fmt.Println("Prijavljen!")

	// 2. Kreiraj ili pridruži se mreži
	choice = prompt(reader, "(1) Kreiraj mrežu  (2) Pridruži se: ")

	var virtualIP, networkID string
	isCreator := choice == "1"

	if isCreator {
		resp, err := apiPost(serverURL+"/network/create", token, nil)
		if err != nil || resp.StatusCode != 201 {
			log.Fatal("Greška pri kreiranju mreže")
		}
		var createResp map[string]string
		json.NewDecoder(resp.Body).Decode(&createResp)
		networkID = createResp["networkID"]
		virtualIP = createResp["virtualIP"]
		fmt.Printf("Mreža kreirana! Kod: %s\n", networkID)
		fmt.Printf("Podijeli ovaj kod s prijateljem.\n")
	} else {
		networkID = prompt(reader, "Upiši kod mreže: ")
		resp, err := apiPost(serverURL+"/network/join", token, map[string]string{
			"networkID": networkID,
		})
		if err != nil || resp.StatusCode != 200 {
			log.Fatal("Greška pri pridruživanju mreži")
		}
		var joinResp struct {
			VirtualIP string           `json:"virtualIP"`
			Peers     []map[string]any `json:"peers"`
		}
		json.NewDecoder(resp.Body).Decode(&joinResp)
		virtualIP = joinResp.VirtualIP
		fmt.Printf("Pridružen mreži %s!\n", networkID)
		for _, p := range joinResp.Peers {
			fmt.Printf("  Peer: %s → %s\n", p["username"], p["virtualIP"])
		}
	}
	fmt.Printf("Tvoja virtualna IP: %s\n", virtualIP)

	// 3. Kreiraj TUN adapter
	dev, err := tun.CreateTUN("vlan0", 1500)
	if err != nil {
		log.Fatal("TUN error:", err)
	}
	name, _ := dev.Name()
	log.Println("TUN adapter kreiran:", name)

	cmd := exec.Command("netsh", "interface", "ip", "set", "address",
		"name=vlan0", "source=static", "addr="+virtualIP, "mask=255.255.255.0")
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Fatal("Netsh error:", string(out), err)
	}
	log.Println("IP adresa:", virtualIP)

	// 4. Otvori UDP socket na random portu
	udpConn, err := net.ListenPacket("udp4", ":0")
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

	// 6. Objavi svoj UDP endpoint (samo jednom)
	wsConn.WriteJSON(map[string]any{
		"type": "announce",
		"data": map[string]any{
			"virtualIP": virtualIP,
			"host":      "127.0.0.1",
			"port":      localPort,
		},
	})
	log.Println("Announce poslан")

	// 7. Čekaj da se peer pojavi
	fmt.Println("Čekam peera...")
	var peerAddr *net.UDPAddr

	for {
		var raw struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}
		if err := wsConn.ReadJSON(&raw); err != nil {
			log.Fatal("WebSocket read error:", err)
		}

		switch raw.Type {
		case "peer_announce":
			var data map[string]any
			json.Unmarshal(raw.Data, &data)
			peerHost := data["host"].(string)
			peerPort := int(data["port"].(float64))
			peerUser := data["username"].(string)
			peerAddr, _ = net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", peerHost, peerPort))
			log.Printf("Peer pronađen: %s na %s\n", peerUser, peerAddr)
		case "peer_joined":
			var data map[string]any
			json.Unmarshal(raw.Data, &data)
			log.Printf("Peer %s se pridružio, čekam announce...\n", data["username"])
		case "peers_list":
			log.Println("Primljena lista peerova")
		}

		if peerAddr != nil {
			break
		}
	}

	// 8. Noise handshake
	cs := noise.NewCipherSuite(noise.DH25519, noise.CipherChaChaPoly, noise.HashBLAKE2b)
	var sendCipher, recvCipher *noise.CipherState

	if isCreator {
		// Creator je responder — čeka handshake
		hs, _ := noise.NewHandshakeState(noise.Config{
			CipherSuite: cs,
			Pattern:     noise.HandshakeNN,
			Initiator:   false,
		})
		log.Println("Čekam handshake...")
		buf := make([]byte, 1500)
		n, _, err := udpConn.ReadFrom(buf)
		if err != nil {
			log.Fatal(err)
		}
		_, _, _, err = hs.ReadMessage(nil, buf[:n])
		if err != nil {
			log.Fatal("Handshake error:", err)
		}
		msg, cs1, cs2, err := hs.WriteMessage(nil, nil)
		if err != nil {
			log.Fatal("Handshake error:", err)
		}
		udpConn.WriteTo(msg, peerAddr)
		recvCipher = cs1
		sendCipher = cs2
	} else {
		// Joiner je initiator — šalje prvi
		hs, _ := noise.NewHandshakeState(noise.Config{
			CipherSuite: cs,
			Pattern:     noise.HandshakeNN,
			Initiator:   true,
		})
		log.Println("Šaljem handshake...")
		time.Sleep(500 * time.Millisecond)
		msg, _, _, err := hs.WriteMessage(nil, nil)
		if err != nil {
			log.Fatal("Handshake error:", err)
		}
		udpConn.WriteTo(msg, peerAddr)

		buf := make([]byte, 1500)
		n, _, err := udpConn.ReadFrom(buf)
		if err != nil {
			log.Fatal(err)
		}
		_, cs1, cs2, err := hs.ReadMessage(nil, buf[:n])
		if err != nil {
			log.Fatal("Handshake error:", err)
		}
		sendCipher = cs1
		recvCipher = cs2
	}

	log.Println("Enkripcija uspostavljena!")

	// 9. TUN → UDP (enkriptirano)
	go func() {
		bufs := make([][]byte, 1)
		bufs[0] = make([]byte, 1500)
		sizes := make([]int, 1)
		for {
			n, err := dev.Read(bufs, sizes, 0)
			if err != nil {
				log.Fatal(err)
			}
			for i := 0; i < n; i++ {
				encrypted, err := sendCipher.Encrypt(nil, nil, bufs[i][:sizes[i]])
				if err != nil {
					continue
				}
				udpConn.WriteTo(encrypted, peerAddr)
			}
		}
	}()

	// 10. UDP → TUN (dekriptirano)
	go func() {
		buf := make([]byte, 2000)
		for {
			n, _, err := udpConn.ReadFrom(buf)
			if err != nil {
				log.Fatal(err)
			}
			decrypted, err := recvCipher.Decrypt(nil, nil, buf[:n])
			if err != nil {
				continue
			}
			dev.Write([][]byte{decrypted}, 0)
		}
	}()

	// 11. Drži program aktivnim
	fmt.Println("=============================")
	fmt.Println("  Tunel aktivan! Pritisni Enter za izlaz.")
	fmt.Println("=============================")
	reader.ReadString('\n')
}
