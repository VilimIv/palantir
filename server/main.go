package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/bcrypt"
)

// Generiraj kriptografski siguran random ključ
func generateSecret() []byte {
	key := make([]byte, 32)
	rand.Read(key)
	return key
}

var jwtSecret = func() []byte {
	// Provjeri environment varijablu
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		decoded, err := hex.DecodeString(secret)
		if err == nil {
			return decoded
		}
	}
	// Ako nema env varijable, generiraj nasumično
	secret := generateSecret()
	log.Printf("UPOZORENJE: JWT_SECRET nije postavljen. Korištenje: set JWT_SECRET=%s\n", hex.EncodeToString(secret))
	return secret
}()

// --- Modeli ---

type User struct {
	Username string
	Password string
}

type Peer struct {
	Username  string `json:"username"`
	VirtualIP string `json:"virtualIP"`
}

type Network struct {
	ID     string
	Peers  []Peer
	nextIP int
}

func (n *Network) nextVirtualIP() string {
	n.nextIP++
	return fmt.Sprintf("10.0.0.%d", n.nextIP)
}

// WebSocket poruka koju server šalje klijentima
type WSMessage struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// Spojeni WebSocket klijent
type WSClient struct {
	Username  string
	NetworkID string
	Conn      *websocket.Conn
	mu        sync.Mutex
}

func (c *WSClient) Send(msg WSMessage) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Conn.WriteJSON(msg)
}

// --- Storage ---

var (
	users     = map[string]*User{}
	networks  = map[string]*Network{}
	clients   = map[string][]*WSClient{}               // networkID → spojeni klijenti
	announces = map[string]map[string]map[string]any{} // networkID → username → {host, port, virtualIP}
	mu        sync.Mutex
)

// --- Pomoćne funkcije ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func generateToken(username string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": username,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	})
	return token.SignedString(jwtSecret)
}

func parseTokenString(tokenStr string) (string, bool) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return "", false
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", false
	}
	username, ok := claims["sub"].(string)
	return username, ok
}

func parseToken(r *http.Request) (string, bool) {
	header := r.Header.Get("Authorization")
	if len(header) < 8 || header[:7] != "Bearer " {
		return "", false
	}
	return parseTokenString(header[7:])
}

// --- Handleri ---

func handleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" || req.Password == "" {
		writeError(w, 400, "Username i password su obavezni")
		return
	}

	mu.Lock()
	defer mu.Unlock()

	if _, exists := users[req.Username]; exists {
		writeError(w, 409, "Korisnik već postoji")
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	users[req.Username] = &User{Username: req.Username, Password: string(hash)}
	writeJSON(w, 201, map[string]string{"message": "Registracija uspješna"})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "Neispravan zahtjev")
		return
	}

	mu.Lock()
	user, exists := users[req.Username]
	mu.Unlock()

	if !exists || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) != nil {
		writeError(w, 401, "Krivi username ili password")
		return
	}

	token, _ := generateToken(req.Username)
	writeJSON(w, 200, map[string]string{"token": token})
}

func handleCreateNetwork(w http.ResponseWriter, r *http.Request) {
	username, ok := parseToken(r)
	if !ok {
		writeError(w, 401, "Niste prijavljeni")
		return
	}

	mu.Lock()
	defer mu.Unlock()

	networkID := fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	network := &Network{ID: networkID}
	ip := network.nextVirtualIP()
	network.Peers = append(network.Peers, Peer{Username: username, VirtualIP: ip})
	networks[networkID] = network

	log.Printf("Mreža %s kreirana, %s → %s\n", networkID, username, ip)
	writeJSON(w, 201, map[string]string{
		"networkID": networkID,
		"virtualIP": ip,
	})
}

func handleJoinNetwork(w http.ResponseWriter, r *http.Request) {
	username, ok := parseToken(r)
	if !ok {
		writeError(w, 401, "Niste prijavljeni")
		return
	}

	var req struct {
		NetworkID string `json:"networkID"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.NetworkID == "" {
		writeError(w, 400, "networkID je obavezan")
		return
	}

	mu.Lock()
	defer mu.Unlock()

	network, exists := networks[req.NetworkID]
	if !exists {
		writeError(w, 404, "Mreža ne postoji")
		return
	}

	ip := network.nextVirtualIP()
	existingPeers := make([]Peer, len(network.Peers))
	copy(existingPeers, network.Peers)
	newPeer := Peer{Username: username, VirtualIP: ip}
	network.Peers = append(network.Peers, newPeer)

	// Obavijesti sve spojene klijente da je novi peer ušao
	for _, client := range clients[req.NetworkID] {
		if client.Username != username {
			client.Send(WSMessage{
				Type: "peer_joined",
				Data: newPeer,
			})
		}
	}

	log.Printf("Mreža %s: %s se pridružio → %s\n", req.NetworkID, username, ip)
	writeJSON(w, 200, map[string]any{
		"virtualIP": ip,
		"peers":     existingPeers,
	})
}

// WebSocket endpoint
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Autentikacija preko query parametra: /ws?token=XXX&networkID=YYY
	tokenStr := r.URL.Query().Get("token")
	networkID := r.URL.Query().Get("networkID")

	username, ok := parseTokenString(tokenStr)
	if !ok {
		writeError(w, 401, "Niste prijavljeni")
		return
	}

	mu.Lock()
	_, networkExists := networks[networkID]
	mu.Unlock()

	if !networkExists {
		writeError(w, 404, "Mreža ne postoji")
		return
	}

	// Upgrade na WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	client := &WSClient{
		Username:  username,
		NetworkID: networkID,
		Conn:      conn,
	}

	mu.Lock()
	clients[networkID] = append(clients[networkID], client)
	mu.Unlock()

	log.Printf("WebSocket: %s spojen na mrežu %s\n", username, networkID)

	// Pošalji trenutnu listu peerova
	mu.Lock()
	peers := networks[networkID].Peers
	mu.Unlock()
	client.Send(WSMessage{Type: "peers_list", Data: peers})

	// Pošalji sve poznate announce-e
	mu.Lock()
	if networkAnnounces, exists := announces[networkID]; exists {
		for announceUsername, data := range networkAnnounces {
			if announceUsername != username {
				client.Send(WSMessage{
					Type: "peer_announce",
					Data: map[string]any{
						"username":  announceUsername,
						"virtualIP": data["virtualIP"],
						"host":      data["host"],
						"port":      data["port"],
					},
				})
			}
		}
	}
	mu.Unlock()

	// Čitaj poruke od klijenta i prosljeđuj peerovima
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		var wsMsg struct {
			Type string         `json:"type"`
			Data map[string]any `json:"data"`
		}
		if json.Unmarshal(msg, &wsMsg) != nil {
			continue
		}
		if wsMsg.Type == "announce" {
			mu.Lock()
			// Spremi announce u memoriju
			if announces[networkID] == nil {
				announces[networkID] = make(map[string]map[string]any)
			}
			announces[networkID][username] = map[string]any{
				"virtualIP": wsMsg.Data["virtualIP"],
				"host":      wsMsg.Data["host"],
				"port":      wsMsg.Data["port"],
			}
			// Proslijedi ostalima
			for _, c := range clients[networkID] {
				if c.Username != username {
					c.Send(WSMessage{
						Type: "peer_announce",
						Data: map[string]any{
							"username":  username,
							"virtualIP": wsMsg.Data["virtualIP"],
							"host":      wsMsg.Data["host"],
							"port":      wsMsg.Data["port"],
						},
					})
				}
			}
			mu.Unlock()
			log.Printf("Announce: %s → %v\n", username, wsMsg.Data)
		}
	}

	// --- Klijent se odspojio ---
	mu.Lock()

	// Ukloni iz liste klijenata
	for i, c := range clients[networkID] {
		if c == client {
			clients[networkID] = append(clients[networkID][:i], clients[networkID][i+1:]...)
			break
		}
	}

	// Pronađi virtualIP i ukloni iz network.Peers
	var leftVirtualIP string
	if net, exists := networks[networkID]; exists {
		for i, p := range net.Peers {
			if p.Username == username {
				leftVirtualIP = p.VirtualIP
				net.Peers = append(net.Peers[:i], net.Peers[i+1:]...)
				break
			}
		}
	}

	// Obriši announce
	if announces[networkID] != nil {
		delete(announces[networkID], username)
	}

	// Kopiraj listu klijenata za notifikaciju (da ne šaljemo pod lockom)
	toNotify := make([]*WSClient, len(clients[networkID]))
	copy(toNotify, clients[networkID])

	mu.Unlock()

	log.Printf("WebSocket: %s odspojio se s mreže %s\n", username, networkID)

	// Obavijesti ostale peere (izvan locka)
	msg := WSMessage{
		Type: "peer_left",
		Data: map[string]string{
			"username":  username,
			"virtualIP": leftVirtualIP,
		},
	}
	for _, c := range toNotify {
		c.Send(msg)
	}
}

func main() {
	http.HandleFunc("/register", handleRegister)
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/network/create", handleCreateNetwork)
	http.HandleFunc("/network/join", handleJoinNetwork)
	http.HandleFunc("/ws", handleWebSocket)

	log.Println("Server pokrenut na :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
