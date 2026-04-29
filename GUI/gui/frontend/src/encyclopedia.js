export const encyclopedia = {
  hr: {
    "Osnove mreže": [
      {
        title: "IP adresa",
        description: `Internet Protocol adresa je jedinstveni numerički identifikator koji svaki uređaj na mreži mora imati da bi mogao komunicirati. Funkcionira poput poštanske adrese — bez nje, paketi podataka ne znaju kamo ići.

Postoje dvije kategorije:
• Privatne adrese — koriste se unutar lokalne mreže (npr. 192.168.1.x, 10.x.x.x). Ove adrese nisu vidljive na internetu i mogu se ponavljati u različitim mrežama.
• Javne adrese — jedinstvene na cijelom internetu, dodjeljuje ih ISP (pružatelj internetskih usluga).

IPv4 koristi 32-bitne adrese (4 broja odvojena točkama, npr. 192.168.1.15), što daje ~4.3 milijarde mogućih adresa. Zbog nedostatka adresa, uveden je IPv6 s 128-bitnim adresama.`,
        diagram: `  Privatna mreža                    Internet
  ┌─────────────────┐              ┌──────────┐
  │ PC: 192.168.1.5 │──┐           │          │
  │ Mob: 192.168.1.8│──┤──Router──▶│ Javni IP │
  │ TV: 192.168.1.20│──┘  NAT     │85.10.44.2│
  └─────────────────┘              └──────────┘
  Svi dijele jednu javnu adresu`,
        inPalantir: "PaLANtir dodjeljuje virtualnu privatnu adresu (10.0.0.x) svakom korisniku. Operativni sustav tretira tu adresu kao da je korisnik na lokalnoj mreži, iako je fizički na drugom kraju svijeta."
      },
      {
        title: "TCP (Transmission Control Protocol)",
        description: `TCP je pouzdani transportni protokol koji garantira da će svi podaci stići na odredište u ispravnom redoslijedu, bez gubitaka. Prije slanja podataka, TCP uspostavlja vezu pomoću trostrukog rukovanja (three-way handshake):

1. Klijent šalje SYN (synchronize) — "Želim uspostaviti vezu"
2. Server odgovara SYN-ACK — "Primljeno, i ja želim"
3. Klijent šalje ACK — "Potvrđujem, veza uspostavljena"

Nakon toga, svaki poslani paket mora biti potvrđen (ACK). Ako potvrda ne stigne, paket se retransmitira. TCP također kontrolira brzinu slanja (flow control) da ne preplavi primatelja.

Prednosti: pouzdanost, garancija redoslijeda, detekcija grešaka.
Mane: veći overhead, viši latency zbog potvrda i retransmisija.`,
        diagram: `  Klijent                    Server
     │                          │
     │──── SYN ────────────────▶│
     │                          │
     │◀──── SYN-ACK ───────────│
     │                          │
     │──── ACK ────────────────▶│
     │                          │
     │    Veza uspostavljena    │
     │                          │
     │──── Podaci + SEQ=1 ────▶│
     │◀──── ACK=2 ─────────────│
     │──── Podaci + SEQ=2 ────▶│
     │◀──── ACK=3 ─────────────│`,
        inPalantir: "PaLANtir koristi TCP za komunikaciju sa serverom — HTTP API pozivi (registracija, login) i WebSocket (signalizacija). Ovi podaci moraju stići pouzdano."
      },
      {
        title: "UDP (User Datagram Protocol)",
        description: `UDP je brzi transportni protokol koji šalje pakete (datagrame) bez uspostave veze i bez garancije dostave. Nema handshakea, nema potvrda, nema retransmisije.

Svaki UDP paket je neovisan — šalje se i zaboravlja. Ako se izgubi, nema automatskog ponovnog slanja. Paketi mogu stići u drugačijem redoslijedu nego što su poslani.

Zašto se onda koristi? Jer je brz. Bez overheada potvrda i retransmisija, UDP ima minimalan latency. Za aplikacije gdje je brzina važnija od savršenosti (gaming, video pozivi, DNS), UDP je idealan izbor.

U gamingu, ako se izgubi jedan frame, bolje je prikazati sljedeći nego čekati retransmisiju starog. Igrač neće primijetiti jedan izgubljen frame, ali HOĆE primijetiti 200ms lag.`,
        diagram: `  TCP (pouzdano, sporije):
  Pošalji → Čekaj potvrdu → Pošalji → Čekaj...
  ████░░░░████░░░░████░░░░████

  UDP (brzo, nepouzdano):
  Pošalji → Pošalji → Pošalji → Pošalji
  ████████████████████████████
  (poneki paket se izgubi, ali promet teče)`,
        inPalantir: "Sav tunel promet između igrača ide preko UDP-a. AES-256-GCM enkripcija s explicit nonce-ovima znači da gubitak pojedinog paketa ne utječe na dekripciju budućih paketa."
      },
      {
        title: "LAN (Local Area Network)",
        description: `LAN je lokalna mreža koja povezuje uređaje na malom geografskom području — tipično unutar jedne kuće, stana ili ureda. Svi uređaji na LAN-u mogu komunicirati direktno, bez prolaska kroz internet.

Karakteristike LAN-a:
• Nizak latency (< 1ms)
• Visoka propusnost (100 Mbps – 10 Gbps)
• Broadcast — poruka poslana svima istovremeno
• Privatni IP adresni prostor

LAN igre koriste broadcast mehanizam za otkrivanje servera. Kad igrač pokrene server, on periodički šalje broadcast poruku "Ja sam ovdje!". Ostali igrači primaju tu poruku i vide server u izborniku igre.

Ovo ne radi preko interneta jer broadcast ne prolazi kroz routere — ostaje unutar LAN-a. Zato su potrebna rješenja poput PaLANtir-a.`,
        inPalantir: "PaLANtir simulira LAN mrežu virtualnim TUN adapterom. Igre vide vlan0 adapter s adresom 10.0.0.x i misle da su svi igrači na istoj fizičkoj mreži. Broadcast paketi se prosljeđuju svim peerovima."
      },
      {
        title: "Subnet i subnet maska",
        description: `Subnet (podmreža) je logička podjela veće mreže na manje segmente. Subnet maska određuje koji dio IP adrese identificira mrežu, a koji uređaj.

Primjer s /24 (255.255.255.0):
  IP:     10.0.0.5
  Maska:  255.255.255.0
  Mreža:  10.0.0.___  (prvih 24 bita)
  Uređaj: _______.5   (zadnjih 8 bitova)

To znači da svi uređaji 10.0.0.1 do 10.0.0.254 pripadaju istoj mreži i mogu komunicirati direktno. Adresa 10.0.0.0 je mrežna adresa, 10.0.0.255 je broadcast.

Česte subnet maske:
  /8  = 255.0.0.0       = 16 mil. adresa
  /16 = 255.255.0.0     = 65.534 adresa
  /24 = 255.255.255.0   = 254 adresa
  /32 = 255.255.255.255 = 1 adresa`,
        inPalantir: "PaLANtir koristi subnet 10.0.0.0/24 za virtualnu mrežu. To podržava do 254 igrača u jednoj sesiji. Subnet maska se automatski postavlja pri konfiguraciji TUN adaptera."
      }
    ],

    "NAT i privatne mreže": [
      {
        title: "NAT (Network Address Translation)",
        description: `NAT je tehnika koja omogućuje da više uređaja dijeli jednu javnu IP adresu. Gotovo svaki kućni router koristi NAT.

Kako radi:
1. PC (192.168.1.5) šalje paket na Google (142.250.74.46)
2. Router zamijeni izvorišnu adresu: 192.168.1.5 → 85.10.44.2 (javni IP)
3. Router zapamti mapping: 192.168.1.5:54321 ↔ 85.10.44.2:48000
4. Google odgovara na 85.10.44.2:48000
5. Router prosljeđuje odgovor natrag na 192.168.1.5:54321

Problem za P2P: ako dva uređaja iza NAT-a žele komunicirati, nijedan ne može inicirati vezu jer su oba "skrivena" iza routera. Router odbija dolazne pakete koji ne pripadaju postojećem mappingu.

Ovo je fundamentalni razlog zašto LAN igre ne rade direktno preko interneta.`,
        diagram: `  PC (192.168.1.5:54321)
     │
     ▼
  ┌─────────┐    NAT tablica:
  │  Router  │    192.168.1.5:54321 ↔ 85.10.44.2:48000
  │   NAT    │    192.168.1.8:12345 ↔ 85.10.44.2:48001
  └────┬─────┘
       │  85.10.44.2:48000
       ▼
   Internet → Google (142.250.74.46)

  Dolazni paketi BEZ mappinga → ODBIJENI!`,
        inPalantir: "PaLANtir zaobilazi NAT koristeći STUN za otkrivanje javne adrese i UDP hole punching za uspostavu direktne veze. Kad to ne uspije, koristi relay server."
      },
      {
        title: "Symmetric NAT",
        description: `Symmetric NAT je najrestriktivniji tip NAT-a. Za svaku različitu destinaciju, router kreira POTPUNO NOVI mapping s DRUGIM vanjskim portom.

Primjer:
  PC šalje na Google → Router mapira: port 48000
  PC šalje na STUN  → Router mapira: port 48001 (DRUGI port!)
  PC šalje na Peera  → Router mapira: port 48002 (OPET DRUGI!)

Problem za hole punching: STUN server nam kaže "tvoja javna adresa je 85.10.44.2:48001". Mi to kažemo peeru. Ali kad peer pošalje paket na :48001, router ga odbija jer je taj port bio za STUN server, ne za peera. Pravi port za peera bi bio :48002, ali to nitko ne zna unaprijed.

Gdje se javlja: mobilni hotspotovi, korporativne mreže, neka sveučilišta.`,
        diagram: `  PC (192.168.1.5)
     │
  ┌──┴──────────────┐
  │  Symmetric NAT  │
  └──┬──────────────┘
     │
     ├──▶ Google    → 85.10.44.2:48000
     ├──▶ STUN      → 85.10.44.2:48001  (DRUGI port!)
     └──▶ Peer      → 85.10.44.2:48002  (OPET DRUGI!)

  STUN kaže: "Ti si :48001"
  Peer šalje na :48001 → ODBIJENO (taj port je za STUN!)
  Hole punching NE RADI.`,
        inPalantir: "PaLANtir detektira symmetric NAT kad probing ne uspije nakon 10 sekundi. Automatski prelazi na relay server koji prosljeđuje enkriptirane pakete. Latency je veći, ali veza funkcionira."
      },
      {
        title: "Cone NAT (Port Restricted)",
        description: `Cone NAT je blaži tip NAT-a gdje router koristi ISTI vanjski port za svu komunikaciju s istog unutarnjeg porta, neovisno o destinaciji.

Primjer:
  PC šalje na Google → Router mapira: port 48000
  PC šalje na STUN  → Router koristi: port 48000 (ISTI!)
  PC šalje na Peera  → Router koristi: port 48000 (ISTI!)

To znači da kad STUN kaže "tvoja adresa je :48000", taj port radi i za peera. Hole punching funkcionira jer:
1. PC šalje na peerovu STUN adresu → NAT otvara mapping za taj par
2. Peer šalje na naš :48000 → NAT propušta jer mapping postoji

Gdje se javlja: većina kućnih routera, eduroam mreže.`,
        inPalantir: "PaLANtir-ov STUN + probing sustav radi savršeno s cone NAT-om. Probing obično uspije unutar 1-2 sekunde."
      }
    ],

    "Prolaz kroz NAT": [
      {
        title: "STUN",
        description: `STUN (Session Traversal Utilities for NAT) je protokol definiran u RFC 5389 koji omogućuje uređaju da otkrije svoju javnu IP adresu i port — informacije koje su potrebne za P2P komunikaciju.

Kako radi:
1. Klijent šalje STUN Binding Request na javni STUN server
2. STUN server čita izvorišnu adresu paketa (javnu IP:port)
3. Server odgovara s tom adresom u XOR-MAPPED-ADDRESS atributu
4. Klijent sad zna svoju javnu adresu

XOR-MAPPED-ADDRESS koristi XOR s "magic cookie" (0x2112A442) da zaobiđe NAT uređaje koji pokušavaju modificirati adrese u payloadu.

STUN je samo za OTKRIVANJE adrese — ne prosljeđuje promet. Zato je besplatan i postoje javni STUN serveri (stun.l.google.com:19302).`,
        diagram: `  Klijent (192.168.1.5:54321)
     │
     │  1. Binding Request
     │     (prazan, 20 bytea)
     ▼
  ┌──────┐
  │ NAT  │  Mijenja izvor: 85.10.44.2:48000
  └──┬───┘
     │
     ▼
  STUN Server (stun.l.google.com)
     │
     │  2. Binding Response
     │     XOR-MAPPED-ADDRESS:
     │     IP = 85.10.44.2
     │     Port = 48000
     ▼
  Klijent sad zna: "Moja javna adresa je 85.10.44.2:48000"`,
        inPalantir: "PaLANtir implementira vlastiti STUN klijent (~60 linija Go koda). Šalje Binding Request na stun.l.google.com:19302 i parsira XOR-MAPPED-ADDRESS iz odgovora. Koristi ISTI UDP socket kao tunel da osigura da STUN vrati mapping za točno taj port."
      },
      {
        title: "UDP Hole Punching",
        description: `UDP hole punching je tehnika za uspostavu P2P veze kroz dva NAT-a istovremeno. Koristi činjenicu da NAT otvara "rupu" za dolazni promet kad uređaj pošalje odlazni paket.

Postupak:
1. A i B otkrivaju svoje javne adrese pomoću STUN-a
2. A i B razmjene adrese kroz signalizacijski server
3. A šalje UDP paket na B-ovu javnu adresu
   → A-ov NAT kreira mapping: "dopusti pakete od B-ove adrese"
   → Paket stiže do B-ovog NAT-a, ali je odbijen (nema mappinga)
4. B šalje UDP paket na A-ovu javnu adresu
   → B-ov NAT kreira mapping: "dopusti pakete od A-ove adrese"
   → Paket stiže do A-ovog NAT-a i PROLAZI (mapping iz koraka 3!)
5. A-ov sljedeći paket prolazi B-ov NAT (mapping iz koraka 4!)
6. Veza uspostavljena!

Ključ je istovremeno slanje — oba peera moraju slati prije nego što prime.`,
        diagram: `  A (iza NAT-a)                         B (iza NAT-a)
     │                                      │
     │  1. Šalje na B-ovu STUN adresu       │
     │──────────────────────X  (B-ov NAT    │
     │  NAT A: "otvori rupu   odbija jer    │
     │   za B-ovu adresu"     nema mappinga) │
     │                                      │
     │       2. B šalje na A-ovu adresu     │
     │  (A-ov NAT propušta ◀────────────────│
     │   jer mapping postoji!)  NAT B:      │
     │                          "otvori      │
     │  3. A šalje opet        rupu za A"    │
     │──────────────────────▶ (prolazi!)     │
     │                                      │
     │        VEZA USPOSTAVLJENA            │`,
        inPalantir: "PaLANtir-ov probing mehanizam šalje probe pakete na SVE kandidate (lokalni + STUN) svakih 500ms. Oba peera šalju istovremeno, čime se NAT 'rupe' otvaraju na obje strane. Prvi probe koji prođe određuje optimalnu putanju."
      },
      {
        title: "TURN / Relay",
        description: `TURN (Traversal Using Relays around NAT) je fallback mehanizam kad direktna P2P veza nije moguća — tipično kad su OBA peera iza symmetric NAT-a.

Relay server je posrednik koji prima pakete od jednog peera i prosljeđuje ih drugom. Sav promet ide:

  A → Relay server → B

Prednosti:
• Uvijek radi, neovisno o tipu NAT-a
• Enkripcija ostaje end-to-end (server vidi samo šifrirani blob)

Mane:
• Veći latency (dvostruki hop)
• Opterećenje servera (propusnost i procesorsko vrijeme)
• Troškovi hostinga servera

TURN se koristi SAMO kad STUN + hole punching ne uspije — to je zadnja opcija.`,
        diagram: `  Bez relay-a (P2P):
  A ◀════════════════════════▶ B
    ~30ms latency

  S relay-om:
  A ◀══════▶ Server ◀══════▶ B
    ~30ms      +       ~30ms
           = ~60ms ukupno

  Server vidi: [šifrirani blob]
  Server NE vidi: sadržaj paketa`,
        inPalantir: "PaLANtir relay server sluša na UDP portu 8081. Klijenti se registriraju pri pokretanju. Ako probing ne uspije nakon 10 sekundi, klijent automatski prelazi na relay. Log pokazuje [RELAY] umjesto [P2P]."
      }
    ],

    "Virtualne mreže": [
      {
        title: "TUN adapter",
        description: `TUN (network TUNnel) je virtualni mrežni adapter koji operira na Layer 3 (mrežnom sloju) OSI modela. Operativni sustav ga tretira kao pravi mrežni uređaj — može mu dodijeliti IP adresu, rute, i aplikacije mogu slati/primati pakete kroz njega.

Kako radi:
1. Aplikacija (PaLANtir) kreira TUN adapter (npr. "vlan0")
2. OS dodijeli mu IP adresu (10.0.0.1) i subnet masku (/24)
3. OS automatski dodaje rutu: "Paketi za 10.0.0.0/24 idu kroz vlan0"
4. Kad igra pošalje paket na 10.0.0.2, OS ga usmjeri na vlan0
5. PaLANtir čita paket s TUN-a, enkriptira ga, šalje peeru preko UDP-a
6. Peer prima, dekriptira, piše na svoj TUN
7. Peerov OS procesira paket — igra prima podatke

TUN radi s IP paketima (Layer 3). Alternativa je TAP koji radi s Ethernet frame-ovima (Layer 2).`,
        diagram: `  Igra šalje na 10.0.0.2
         │
         ▼
  ┌──────────────┐
  │   OS Kernel  │  Ruting tablica:
  │              │  10.0.0.0/24 → vlan0
  └──────┬───────┘
         │
         ▼
  ┌──────────────┐
  │  TUN "vlan0" │  Virtualni adapter
  └──────┬───────┘
         │
         ▼
  ┌──────────────┐
  │  PaLANtir    │  Čita IP paket
  │  Encrypt     │  AES-256-GCM
  │  Send UDP    │  → peer
  └──────────────┘`,
        inPalantir: "PaLANtir koristi wireguard/tun Go biblioteku za kreiranje TUN adaptera. Na Windowsu koristi Wintun driver, na Linuxu kernel TUN modul. Offset od 16 bytea je potreban na Linuxu za virtio net header."
      },
      {
        title: "VPN vs Virtualni LAN",
        description: `VPN (Virtual Private Network) i virtualni LAN dijele sličnu tehnologiju (enkriptirani tuneli) ali služe različitim svrhama:

VPN:
• Usmjerava SAV internetski promet kroz tunel
• Cilj: privatnost, zaobilaženje geo-restrikcija, pristup korporativnoj mreži
• Obično klijent-server arhitektura
• Primjeri: OpenVPN, WireGuard, NordVPN

Virtualni LAN:
• Kreira virtualnu lokalnu mrežu samo za specifične aplikacije
• Cilj: omogućiti LAN gaming/dijeljenje između udaljenih korisnika
• Obično peer-to-peer arhitektura (full mesh)
• Primjeri: Hamachi, ZeroTier, Radmin VPN, PaLANtir

Ključna razlika: VPN štiti privatnost na internetu, virtualni LAN simulira fizičku blizinu uređaja.`,
        inPalantir: "PaLANtir je virtualni LAN sustav fokusiran na gaming. Ne usmjerava sav promet — samo promet za virtualnu mrežu (10.0.0.0/24). Ostatak interneta radi normalno."
      }
    ],

    "Sigurnost i enkripcija": [
      {
        title: "AES-256-GCM",
        description: `AES-256-GCM je simetrični enkripcijski algoritam koji se smatra zlatnim standardom za zaštitu podataka.

AES (Advanced Encryption Standard):
• Simetrična enkripcija — isti ključ za šifriranje i dešifriranje
• 256-bit ključ — 2^256 mogućih kombinacija (praktički nemoguće probiti brute-forceom)
• Blok šifra — procesira podatke u blokovima od 128 bita

GCM (Galois/Counter Mode):
• Mod rada koji kombinira enkripciju I autentikaciju
• Šifrira podatke (confidentiality)
• Generira autentikacijski tag od 16 bytea (integrity)
• Ako itko promijeni i jedan bit šifriranog paketa, tag se ne poklapa i dekripcija odbija paket

Nonce (Number Used Once):
• 12-byte broj koji mora biti JEDINSTVEN za svako šifriranje s istim ključem
• Ako se nonce ponovi, sigurnost je kompromitirana
• Sekvencijalni nonce: 0, 1, 2, 3... — problem ako se paket izgubi
• Explicit nonce: svaki paket nosi svoj nonce — radi čak i s gubitkom`,
        diagram: `  Enkripcija:
  ┌─────────────────────────────────────────┐
  │ Plaintext  │  AES-256-GCM  │ Ciphertext │
  │ "Hello"    │  + Ključ      │ 0xA3F2...  │
  │            │  + Nonce      │ + Tag 16B  │
  └─────────────────────────────────────────┘

  Paket format u PaLANtiru:
  ┌─────────┬──────────┬───────────────────┐
  │ Counter │ Encrypted│ GCM Auth Tag      │
  │ 8 bytes │ data     │ 16 bytes          │
  └─────────┴──────────┴───────────────────┘
  Counter = explicit nonce (rješava UDP gubitak)`,
        inPalantir: "PaLANtir derivira dva AES-256-GCM ključa iz Noise handshakea (jedan po smjeru). Svaki paket sadrži 8-byte counter kao explicit nonce. Gubitak paketa ne utječe na dekripciju — svaki paket je neovisan."
      },
      {
        title: "Noise Protocol (NN Pattern)",
        description: `Noise Protocol Framework je sustav za kriptografske handshake protokole. Definira kako dva uređaja mogu sigurno razmjenjivati ključeve.

NN pattern ("No authentication, No authentication"):
• Najjednostavniji pattern — nijedan peer nema prethodni identitet
• Pruža: forward secrecy, zaštitu od pasivnog prisluškivanja
• NE pruža: zaštitu od MITM (man-in-the-middle) napada

Handshake tok:
1. Initiator generira ephemeral (privremeni) Diffie-Hellman ključni par
2. Šalje javni dio respondera
3. Responder generira svoj ephemeral par
4. Šalje javni dio + obavlja DH razmjenu
5. Oba peera izračunavaju zajednički tajni ključ

Koriste ga: WireGuard, Lightning Network, WhatsApp (varijante).`,
        diagram: `  Noise NN Handshake:

  Initiator                    Responder
     │                            │
     │  1. e (ephemeral key)      │
     │────────────────────────▶   │
     │    + payload (random 32B)  │
     │                            │
     │  2. e, ee                  │
     │   ◀────────────────────────│
     │    + payload (random 32B)  │
     │                            │
     │  Oba izračunaju:           │
     │  shared_secret = DH(e, e)  │
     │                            │
     │  Deriviraju AES ključeve   │
     │  iz shared_secret          │`,
        inPalantir: "PaLANtir koristi Noise NN za handshake s flynn/noise Go bibliotekom. Tijekom handshakea, oba peera razmjenjuju 32-byte random vrijednosti u payloadu, koje se koriste za derivaciju AES-256-GCM ključeva putem SHA-256."
      },
      {
        title: "JWT (JSON Web Token)",
        description: `JWT je kompaktni token za autentikaciju koji se koristi umjesto sesija. Nakon uspješnog logina, server generira potpisan token koji klijent prilaže svakom budućem zahtjevu.

Struktura JWT-a (3 dijela odvojena točkom):
1. Header — algoritam (HS256) i tip (JWT)
2. Payload — podaci (username, expiration, issued at)
3. Signature — HMAC potpis koji dokazuje autentičnost

Prednosti nad sesijama:
• Stateless — server ne mora čuvati stanje sesije
• Lako provjerljiv — samo treba tajni ključ
• Ima expiration — automatski istječe

Sigurnost: token je potpisan, ne šifriran. Svatko može pročitati payload, ali ne može ga mijenjati bez tajnog ključa.`,
        diagram: `  JWT token:
  eyJhbGci.eyJzdWIi.SflKxwRJ

  ┌──────────┐  ┌──────────────┐  ┌───────────┐
  │ Header   │  │ Payload      │  │ Signature │
  │ alg:HS256│  │ sub:"Vilim"  │  │ HMAC-     │
  │ typ:JWT  │  │ exp:168...   │  │ SHA256    │
  └──────────┘  └──────────────┘  └───────────┘
       │              │                 │
       └──────────────┴─────────────────┘
                    Base64 encoded`,
        inPalantir: "Nakon logina, PaLANtir server izdaje JWT s 24h expiracijom. Klijent ga šalje u Authorization headeru za API pozive i kao query parametar za WebSocket konekciju."
      }
    ],

    "Komunikacija": [
      {
        title: "WebSocket",
        description: `WebSocket je protokol za dvosmjernu real-time komunikaciju. Za razliku od HTTP-a (klijent pita → server odgovara), WebSocket drži konekciju otvorenom i obje strane mogu slati poruke bilo kad.

Uspostava:
1. Klijent šalje HTTP Upgrade zahtjev
2. Server odgovara sa "101 Switching Protocols"
3. Konekcija se "upgradea" u WebSocket
4. Obje strane šalju/primaju poruke slobodno

Prednosti nad pollingom (periodičnim HTTP zahtjevima):
• Niži latency — nema ponovne uspostave konekcije
• Manje overheada — nema HTTP headera za svaku poruku
• Real-time — server može "gurati" podatke odmah`,
        inPalantir: "PaLANtir koristi WebSocket za signalizaciju: primanje announce poruka (IP/port kandidati), obavijesti o novim/odlazeciim peerima (peer_joined, peer_left), i listu peerova pri spajanju."
      },
      {
        title: "Full Mesh mreža",
        description: `Full mesh je mrežna topologija gdje je svaki čvor direktno spojen sa svakim drugim. Za N čvorova postoji N*(N-1)/2 zasebnih veza.

Primjer s 4 igrača:
  4 * 3 / 2 = 6 veza

Prednosti:
• Optimalan latency — svaka komunikacija je direktna
• Nema single point of failure — ako jedan čvor padne, ostali nastavljaju
• Svaki par ima nezavisnu enkripciju

Mane:
• Broj veza raste kvadratno — 10 igrača = 45 veza
• Svaki peer mora održavati stanje za svakog drugog

Za gaming (tipično 2-8 igrača), full mesh je idealan. Za veće mreže (100+ čvorova) koriste se druge topologije.`,
        diagram: `  4 igrača — Full Mesh:

      A ══════════ B
      │╲          ╱│
      │  ╲      ╱  │
      │    ╲  ╱    │
      │     ╳      │
      │   ╱   ╲    │
      │ ╱       ╲  │
      C ══════════ D

  6 nezavisnih tunela,
  svaki sa svojim AES ključevima`,
        inPalantir: "PaLANtir koristi full mesh — svaki par igrača ima zasebni Noise handshake, zasebne AES-256-GCM ključeve, i nezavisne nonce countere. Broadcast se replicira svim peerima pojedinačno."
      }
    ],

    "PaLANtir specifično": [
      {
        title: "Kako PaLANtir radi",
        description: `Kompletan tok od pokretanja do igranja igre:

1. REGISTRACIJA/LOGIN
   Klijent šalje username/password na server. Server hashira lozinku bcrypt-om, vraća JWT token.

2. KREIRANJE/PRIDRUŽIVANJE MREŽI
   Kreator dobije mrežni kod (npr. 857436) i virtualnu IP (10.0.0.1). Drugi igrač upiše kod i dobije sljedeću IP (10.0.0.2).

3. TUN ADAPTER
   Kreira se virtualni mrežni adapter "vlan0" s dodijeljenom IP adresom. OS dodaje rutu za 10.0.0.0/24.

4. STUN OTKRIVANJE
   Klijent kontaktira STUN server i saznaje svoju javnu IP:port adresu.

5. KANDIDATI I ANNOUNCE
   Objavljuje lokalni + STUN kandidat putem WebSocketa. Svi ostali peeri primaju listu kandidata.

6. PROBING (do 10s)
   Šalje probe pakete na sve kandidate paralelno. Prvi odgovor određuje optimalnu putanju. Ako ništa ne uspije → relay fallback.

7. NOISE HANDSHAKE
   Razmjena ključeva putem Noise NN patterna + random payload za AES key derivation.

8. TUNEL AKTIVAN
   Paketi s TUN adaptera se enkriptiraju (AES-256-GCM + explicit nonce), šalju peeru putem UDP-a (direktno ili relay), dekriptiraju, pišu na peerov TUN. Igra vidi "LAN" mrežu.`,
        diagram: `  ┌──────┐   HTTP    ┌──────────┐   HTTP    ┌──────┐
  │  A   │◀────────▶│  Server  │◀────────▶│  B   │
  │      │   WS     │  :8080   │   WS     │      │
  │      │◀────────▶│          │◀────────▶│      │
  └──┬───┘          │  Relay   │          └──┬───┘
     │              │  :8081   │              │
     │              └──────────┘              │
     │                                       │
     │  UDP P2P (enkriptirano)               │
     │◀═════════════════════════════════════▶│
     │  ili UDP Relay (kad P2P ne radi)      │
     │◀════▶ Server :8081 ◀════▶            │
     │                                       │
  ┌──┴───┐                              ┌───┴──┐
  │ TUN  │  vlan0: 10.0.0.1            │ TUN  │  vlan0: 10.0.0.2
  │ Igra │  "LAN mreža"               │ Igra │  "LAN mreža"
  └──────┘                              └──────┘`,
        image: "/images/palantir-saruman.gif",
        inPalantir: "Ovo je cjelokupna arhitektura PaLANtira. Server služi samo za signalizaciju i relay — sav gaming promet ide P2P kad god je moguće."
      },
      {
        title: "Postavljanje vatrozida",
        description: `Da bi PaLANtir i LAN igre radili, potrebno je dopustiti mrežni promet na virtualnoj mreži (10.0.0.0/24).

WINDOWS (PowerShell kao administrator):

1. Dopusti ICMP (ping):
New-NetFirewallRule -DisplayName "Palantir ICMP" -Protocol ICMPv4 -IcmpType 8 -Direction Inbound -Action Allow

2. Dopusti sav promet s virtualne mreže:
New-NetFirewallRule -DisplayName "Palantir VPN All" -Direction Inbound -RemoteAddress 10.0.0.0/24 -Action Allow

3. Uklanjanje pravila kad završiš:
Remove-NetFirewallRule -DisplayName "Palantir ICMP"
Remove-NetFirewallRule -DisplayName "Palantir VPN All"

LINUX (terminal):

1. Dopusti sav promet s virtualne mreže:
sudo iptables -A INPUT -s 10.0.0.0/24 -j ACCEPT

Ili s ufw:
sudo ufw allow from 10.0.0.0/24

NAPOMENA: Ova pravila dopuštaju promet SAMO s virtualne mreže (10.0.0.x). Jedini način da paket ima tu izvorišnu adresu je da dođe kroz PaLANtir-ov enkriptirani tunel. Sigurnosni rizik je minimalan.`,
        inPalantir: "Bez ovih pravila, ping će raditi ali igre neće moći komunicirati. Windows posebno blokira dolazni promet na novim mrežnim adapterima (vlan0 se klasificira kao 'Public' mreža)."
      },
      {
        title: "Poznati problemi i rješenja",
        description: `TUN adapter zahtijeva administratorske ovlasti:
• Windows: pokreni .exe kao Administrator
• Linux: koristi sudo ./palantir

Program se ne može spojiti na server:
• Provjeri je li server pokrenut
• Provjeri je li URL ispravan u kodu
• Provjeri je li port 8080 (TCP) otvoren na serveru

Probe timeout (10 sekundi):
• Oba peera su iza symmetric NAT-a
• PaLANtir automatski prelazi na relay
• Relay dodaje latency ali funkcionira

Igra ne vidi server (LAN discovery):
• Provjeri vatrozid pravila (pogledaj "Postavljanje vatrozida")
• Provjeri rade li pingovi između peerova
• Neki igre trebaju specifične portove otvorene

Visok latency kroz relay:
• Relay dodaje ~30-80ms (ovisno o udaljenosti servera)
• Normalno za symmetric NAT scenarije
• Za bolje performanse, koristite mreže s cone NAT-om`,
        inPalantir: "Većina problema se rješava provjerom vatrozida i osiguravanjem da su oba klijenta na istoj verziji koda."
      }
    ]
  },

  en: {
    "Network Basics": [
      {
        title: "IP Address",
        description: `An Internet Protocol address is a unique numeric identifier that every device on a network must have to communicate. It works like a postal address — without it, data packets don't know where to go.

There are two categories:
• Private addresses — used within local networks (e.g., 192.168.1.x, 10.x.x.x). These are not visible on the internet and can repeat across different networks.
• Public addresses — unique across the entire internet, assigned by ISPs.

IPv4 uses 32-bit addresses (4 numbers separated by dots, e.g., 192.168.1.15), giving ~4.3 billion possible addresses. Due to address exhaustion, IPv6 with 128-bit addresses was introduced.`,
        diagram: `  Private Network                    Internet
  ┌─────────────────┐              ┌──────────┐
  │ PC: 192.168.1.5 │──┐           │          │
  │ Phone: .1.8     │──┤──Router──▶│Public IP │
  │ TV: .1.20       │──┘  NAT     │85.10.44.2│
  └─────────────────┘              └──────────┘
  All share one public address`,
        inPalantir: "PaLANtir assigns a virtual private address (10.0.0.x) to each user. The OS treats it as if the user is on a local network, even though they're physically on the other side of the world."
      },
      {
        title: "TCP (Transmission Control Protocol)",
        description: `TCP is a reliable transport protocol that guarantees all data arrives at the destination in the correct order, without losses. Before sending data, TCP establishes a connection using a three-way handshake:

1. Client sends SYN (synchronize) — "I want to establish a connection"
2. Server replies SYN-ACK — "Received, I want to connect too"
3. Client sends ACK — "Confirmed, connection established"

After that, every sent packet must be acknowledged (ACK). If acknowledgment doesn't arrive, the packet is retransmitted. TCP also controls sending speed (flow control) to not overwhelm the receiver.

Pros: reliability, order guarantee, error detection.
Cons: higher overhead, higher latency due to acknowledgments and retransmissions.`,
        diagram: `  Client                      Server
     │                            │
     │──── SYN ──────────────────▶│
     │                            │
     │◀──── SYN-ACK ─────────────│
     │                            │
     │──── ACK ──────────────────▶│
     │                            │
     │    Connection established  │
     │                            │
     │──── Data + SEQ=1 ────────▶│
     │◀──── ACK=2 ───────────────│
     │──── Data + SEQ=2 ────────▶│
     │◀──── ACK=3 ───────────────│`,
        inPalantir: "PaLANtir uses TCP for server communication — HTTP API calls (registration, login) and WebSocket (signaling). This data must arrive reliably."
      },
      {
        title: "UDP (User Datagram Protocol)",
        description: `UDP is a fast transport protocol that sends packets (datagrams) without establishing a connection and without delivery guarantees. No handshake, no acknowledgments, no retransmission.

Each UDP packet is independent — fire and forget. If one is lost, there's no automatic resending. Packets may arrive in a different order than sent.

Why use it then? Because it's fast. Without the overhead of acknowledgments and retransmissions, UDP has minimal latency. For applications where speed matters more than perfection (gaming, video calls, DNS), UDP is ideal.

In gaming, if one frame is lost, it's better to show the next one than wait for retransmission of the old one. A player won't notice one lost frame, but WILL notice 200ms of lag.`,
        diagram: `  TCP (reliable, slower):
  Send → Wait ACK → Send → Wait...
  ████░░░░████░░░░████░░░░████

  UDP (fast, unreliable):
  Send → Send → Send → Send
  ████████████████████████████
  (some packets lost, but traffic flows)`,
        inPalantir: "All tunnel traffic between players goes over UDP. AES-256-GCM encryption with explicit nonces means that losing a single packet doesn't affect decryption of future packets."
      },
      {
        title: "LAN (Local Area Network)",
        description: `A LAN is a local network connecting devices in a small geographic area — typically within a single home, apartment, or office. All devices on a LAN can communicate directly, without going through the internet.

LAN characteristics:
• Low latency (< 1ms)
• High throughput (100 Mbps – 10 Gbps)
• Broadcast — message sent to everyone simultaneously
• Private IP address space

LAN games use broadcast for server discovery. When a player starts a server, it periodically sends a broadcast message "I'm here!". Other players receive it and see the server in the game menu.

This doesn't work over the internet because broadcast doesn't pass through routers — it stays within the LAN.`,
        inPalantir: "PaLANtir simulates a LAN using a virtual TUN adapter. Games see the vlan0 adapter with address 10.0.0.x and think all players are on the same physical network. Broadcast packets are forwarded to all peers."
      },
      {
        title: "Subnet and Subnet Mask",
        description: `A subnet is a logical division of a larger network into smaller segments. The subnet mask determines which part of an IP address identifies the network and which identifies the device.

Example with /24 (255.255.255.0):
  IP:     10.0.0.5
  Mask:   255.255.255.0
  Network:10.0.0.___  (first 24 bits)
  Device: _______.5   (last 8 bits)

This means all devices 10.0.0.1 through 10.0.0.254 belong to the same network. Address 10.0.0.0 is the network address, 10.0.0.255 is broadcast.

Common subnet masks:
  /8  = 255.0.0.0       = 16 million addresses
  /16 = 255.255.0.0     = 65,534 addresses
  /24 = 255.255.255.0   = 254 addresses`,
        inPalantir: "PaLANtir uses subnet 10.0.0.0/24 for the virtual network. This supports up to 254 players in one session."
      }
    ],

    "NAT and Private Networks": [
      {
        title: "NAT (Network Address Translation)",
        description: `NAT is a technique that allows multiple devices to share a single public IP address. Nearly every home router uses NAT.

How it works:
1. PC (192.168.1.5) sends a packet to Google (142.250.74.46)
2. Router replaces source: 192.168.1.5 → 85.10.44.2 (public IP)
3. Router remembers mapping: 192.168.1.5:54321 ↔ 85.10.44.2:48000
4. Google replies to 85.10.44.2:48000
5. Router forwards response back to 192.168.1.5:54321

The P2P problem: if two devices behind NAT want to communicate, neither can initiate because both are "hidden" behind routers. The router rejects incoming packets that don't belong to an existing mapping.

This is the fundamental reason why LAN games don't work directly over the internet.`,
        diagram: `  PC (192.168.1.5:54321)
     │
     ▼
  ┌─────────┐    NAT table:
  │  Router  │    192.168.1.5:54321 ↔ 85.10.44.2:48000
  │   NAT    │    192.168.1.8:12345 ↔ 85.10.44.2:48001
  └────┬─────┘
       │  85.10.44.2:48000
       ▼
   Internet → Google (142.250.74.46)

  Incoming packets WITHOUT mapping → REJECTED!`,
        inPalantir: "PaLANtir bypasses NAT using STUN for public address discovery and UDP hole punching for establishing direct connections. When that fails, it uses a relay server."
      },
      {
        title: "Symmetric NAT",
        description: `Symmetric NAT is the most restrictive NAT type. For each different destination, the router creates a COMPLETELY NEW mapping with a DIFFERENT external port.

Example:
  PC sends to Google → Router maps: port 48000
  PC sends to STUN  → Router maps: port 48001 (DIFFERENT!)
  PC sends to Peer  → Router maps: port 48002 (DIFFERENT AGAIN!)

Problem for hole punching: STUN tells us "your public address is 85.10.44.2:48001". We tell the peer. But when the peer sends to :48001, the router rejects it because that port was for the STUN server, not the peer.

Found in: mobile hotspots, corporate networks, some universities.`,
        diagram: `  PC (192.168.1.5)
     │
  ┌──┴──────────────┐
  │  Symmetric NAT  │
  └──┬──────────────┘
     │
     ├──▶ Google → 85.10.44.2:48000
     ├──▶ STUN   → 85.10.44.2:48001  (DIFFERENT!)
     └──▶ Peer   → 85.10.44.2:48002  (DIFFERENT AGAIN!)

  STUN says: "You are :48001"
  Peer sends to :48001 → REJECTED!
  Hole punching FAILS.`,
        inPalantir: "PaLANtir detects symmetric NAT when probing fails after 10 seconds. It automatically switches to the relay server. Latency is higher, but the connection works."
      },
      {
        title: "Cone NAT (Port Restricted)",
        description: `Cone NAT is a more lenient NAT type where the router uses the SAME external port for all communication from the same internal port, regardless of destination.

Example:
  PC sends to Google → Router maps: port 48000
  PC sends to STUN  → Router uses: port 48000 (SAME!)
  PC sends to Peer  → Router uses: port 48000 (SAME!)

This means when STUN says "your address is :48000", that port works for the peer too. Hole punching works because:
1. PC sends to peer's STUN address → NAT opens mapping for that pair
2. Peer sends to our :48000 → NAT allows it because mapping exists

Found in: most home routers, eduroam networks.`,
        inPalantir: "PaLANtir's STUN + probing system works perfectly with cone NAT. Probing usually succeeds within 1-2 seconds."
      }
    ],

    "NAT Traversal": [
      {
        title: "STUN",
        description: `STUN (Session Traversal Utilities for NAT) is a protocol defined in RFC 5389 that allows a device to discover its public IP address and port — information needed for P2P communication.

How it works:
1. Client sends STUN Binding Request to a public STUN server
2. STUN server reads the source address of the packet (public IP:port)
3. Server responds with that address in XOR-MAPPED-ADDRESS attribute
4. Client now knows its public address

XOR-MAPPED-ADDRESS uses XOR with a "magic cookie" (0x2112A442) to bypass NAT devices that try to modify addresses in payloads.

STUN only DISCOVERS addresses — it doesn't relay traffic. That's why it's free and public servers exist (stun.l.google.com:19302).`,
        diagram: `  Client (192.168.1.5:54321)
     │
     │  1. Binding Request
     │     (empty, 20 bytes)
     ▼
  ┌──────┐
  │ NAT  │  Changes source: 85.10.44.2:48000
  └──┬───┘
     │
     ▼
  STUN Server (stun.l.google.com)
     │
     │  2. Binding Response
     │     XOR-MAPPED-ADDRESS:
     │     IP = 85.10.44.2
     │     Port = 48000
     ▼
  Client knows: "My public address is 85.10.44.2:48000"`,
        inPalantir: "PaLANtir implements its own STUN client (~60 lines of Go). It sends a Binding Request to stun.l.google.com:19302 and parses XOR-MAPPED-ADDRESS from the response. Uses the SAME UDP socket as the tunnel."
      },
      {
        title: "UDP Hole Punching",
        description: `UDP hole punching is a technique for establishing P2P connections through two NATs simultaneously. It exploits the fact that NAT opens a "hole" for incoming traffic when a device sends an outgoing packet.

Process:
1. A and B discover their public addresses via STUN
2. A and B exchange addresses through signaling server
3. A sends UDP to B's public address
   → A's NAT creates mapping: "allow packets from B's address"
   → Packet reaches B's NAT but is REJECTED (no mapping)
4. B sends UDP to A's public address
   → B's NAT creates mapping: "allow packets from A's address"
   → Packet reaches A's NAT and PASSES (mapping from step 3!)
5. A's next packet passes B's NAT (mapping from step 4!)
6. Connection established!

The key is simultaneous sending — both peers must send before receiving.`,
        diagram: `  A (behind NAT)                      B (behind NAT)
     │                                      │
     │  1. Send to B's STUN address         │
     │──────────────────────X  (B's NAT     │
     │  NAT A: "open hole    rejects, no    │
     │   for B's address"    mapping)       │
     │                                      │
     │       2. B sends to A's address      │
     │  (A's NAT allows  ◀─────────────────│
     │   mapping exists!)   NAT B:         │
     │                      "open hole     │
     │  3. A sends again     for A"         │
     │──────────────────────▶ (passes!)     │
     │                                      │
     │       CONNECTION ESTABLISHED         │`,
        inPalantir: "PaLANtir's probing mechanism sends probe packets to ALL candidates (local + STUN) every 500ms. Both peers send simultaneously, opening NAT 'holes' on both sides."
      },
      {
        title: "TURN / Relay",
        description: `TURN (Traversal Using Relays around NAT) is a fallback mechanism when direct P2P isn't possible — typically when BOTH peers are behind symmetric NAT.

The relay server acts as an intermediary that receives packets from one peer and forwards them to the other:
  A → Relay server → B

Pros:
• Always works regardless of NAT type
• Encryption remains end-to-end (server only sees encrypted blob)

Cons:
• Higher latency (double hop)
• Server load (bandwidth and CPU)
• Hosting costs

TURN is used ONLY when STUN + hole punching fails — it's the last resort.`,
        diagram: `  Without relay (P2P):
  A ◀════════════════════════▶ B
    ~30ms latency

  With relay:
  A ◀══════▶ Server ◀══════▶ B
    ~30ms      +       ~30ms
           = ~60ms total

  Server sees: [encrypted blob]
  Server CANNOT see: packet contents`,
        inPalantir: "PaLANtir's relay server listens on UDP port 8081. Clients register at startup. If probing fails after 10 seconds, the client automatically switches to relay."
      }
    ],

    "Virtual Networks": [
      {
        title: "TUN Adapter",
        description: `TUN (network TUNnel) is a virtual network adapter operating at Layer 3 (network layer) of the OSI model. The OS treats it as a real network device — it can be assigned an IP address, routes, and applications can send/receive packets through it.

How it works:
1. Application (PaLANtir) creates TUN adapter (e.g., "vlan0")
2. OS assigns IP address (10.0.0.1) and subnet mask (/24)
3. OS automatically adds route: "Packets for 10.0.0.0/24 go through vlan0"
4. When a game sends a packet to 10.0.0.2, OS routes it to vlan0
5. PaLANtir reads the packet from TUN, encrypts it, sends to peer via UDP
6. Peer receives, decrypts, writes to their TUN
7. Peer's OS processes packet — game receives data`,
        diagram: `  Game sends to 10.0.0.2
         │
         ▼
  ┌──────────────┐
  │   OS Kernel  │  Routing table:
  │              │  10.0.0.0/24 → vlan0
  └──────┬───────┘
         │
         ▼
  ┌──────────────┐
  │  TUN "vlan0" │  Virtual adapter
  └──────┬───────┘
         │
         ▼
  ┌──────────────┐
  │  PaLANtir    │  Reads IP packet
  │  Encrypt     │  AES-256-GCM
  │  Send UDP    │  → peer
  └──────────────┘`,
        inPalantir: "PaLANtir uses the wireguard/tun Go library to create the TUN adapter. On Windows it uses the Wintun driver, on Linux the kernel TUN module. A 16-byte offset is required on Linux for the virtio net header."
      },
      {
        title: "VPN vs Virtual LAN",
        description: `VPN (Virtual Private Network) and virtual LAN share similar technology (encrypted tunnels) but serve different purposes:

VPN:
• Routes ALL internet traffic through the tunnel
• Goal: privacy, bypassing geo-restrictions, corporate network access
• Usually client-server architecture
• Examples: OpenVPN, WireGuard, NordVPN

Virtual LAN:
• Creates a virtual local network only for specific applications
• Goal: enable LAN gaming/sharing between remote users
• Usually peer-to-peer architecture (full mesh)
• Examples: Hamachi, ZeroTier, Radmin VPN, PaLANtir

Key difference: VPN protects internet privacy, virtual LAN simulates physical proximity of devices.`,
        inPalantir: "PaLANtir is a virtual LAN system focused on gaming. It doesn't route all traffic — only traffic for the virtual network (10.0.0.0/24). The rest of the internet works normally."
      }
    ],

    "Security and Encryption": [
      {
        title: "AES-256-GCM",
        description: `AES-256-GCM is a symmetric encryption algorithm considered the gold standard for data protection.

AES (Advanced Encryption Standard):
• Symmetric — same key for encryption and decryption
• 256-bit key — 2^256 possible combinations (practically impossible to brute-force)
• Block cipher — processes data in 128-bit blocks

GCM (Galois/Counter Mode):
• Combines encryption AND authentication
• Encrypts data (confidentiality)
• Generates 16-byte authentication tag (integrity)
• If anyone modifies even one bit, the tag won't match and decryption rejects the packet

Nonce (Number Used Once):
• 12-byte number that must be UNIQUE for every encryption with the same key
• Sequential nonce: 0, 1, 2, 3... — problem if a packet is lost
• Explicit nonce: each packet carries its own — works even with packet loss`,
        diagram: `  Encryption:
  ┌─────────────────────────────────────────┐
  │ Plaintext  │  AES-256-GCM  │ Ciphertext │
  │ "Hello"    │  + Key        │ 0xA3F2...  │
  │            │  + Nonce      │ + Tag 16B  │
  └─────────────────────────────────────────┘

  PaLANtir packet format:
  ┌─────────┬──────────┬───────────────────┐
  │ Counter │ Encrypted│ GCM Auth Tag      │
  │ 8 bytes │ data     │ 16 bytes          │
  └─────────┴──────────┴───────────────────┘
  Counter = explicit nonce (solves UDP loss)`,
        inPalantir: "PaLANtir derives two AES-256-GCM keys from the Noise handshake (one per direction). Each packet contains an 8-byte counter as explicit nonce. Packet loss doesn't affect decryption."
      },
      {
        title: "Noise Protocol (NN Pattern)",
        description: `The Noise Protocol Framework is a system for cryptographic handshake protocols. It defines how two devices can securely exchange keys.

NN pattern ("No authentication, No authentication"):
• Simplest pattern — neither peer has a prior identity
• Provides: forward secrecy, protection from passive eavesdropping
• Does NOT provide: protection from MITM attacks

Used by: WireGuard, Lightning Network, WhatsApp (variants).`,
        diagram: `  Noise NN Handshake:

  Initiator                    Responder
     │                            │
     │  1. e (ephemeral key)      │
     │────────────────────────▶   │
     │    + payload (random 32B)  │
     │                            │
     │  2. e, ee                  │
     │   ◀────────────────────────│
     │    + payload (random 32B)  │
     │                            │
     │  Both compute:             │
     │  shared_secret = DH(e, e)  │
     │  Derive AES keys           │`,
        inPalantir: "PaLANtir uses Noise NN for handshake with the flynn/noise Go library. During handshake, both peers exchange 32-byte random values in the payload, used for AES-256-GCM key derivation via SHA-256."
      },
      {
        title: "JWT (JSON Web Token)",
        description: `JWT is a compact authentication token used instead of sessions. After successful login, the server generates a signed token that the client attaches to every future request.

JWT structure (3 parts separated by dots):
1. Header — algorithm (HS256) and type (JWT)
2. Payload — data (username, expiration)
3. Signature — HMAC signature proving authenticity

Security: the token is signed, not encrypted. Anyone can read the payload, but cannot modify it without the secret key.`,
        diagram: `  JWT token:
  eyJhbGci.eyJzdWIi.SflKxwRJ

  ┌──────────┐  ┌──────────────┐  ┌───────────┐
  │ Header   │  │ Payload      │  │ Signature │
  │ alg:HS256│  │ sub:"Vilim"  │  │ HMAC-     │
  │ typ:JWT  │  │ exp:168...   │  │ SHA256    │
  └──────────┘  └──────────────┘  └───────────┘`,
        inPalantir: "After login, PaLANtir server issues a JWT with 24h expiration. The client sends it in the Authorization header for API calls and as a query parameter for WebSocket."
      }
    ],

    "Communication": [
      {
        title: "WebSocket",
        description: `WebSocket is a protocol for bidirectional real-time communication. Unlike HTTP (client asks → server answers), WebSocket keeps the connection open and both sides can send messages at any time.

Establishment:
1. Client sends HTTP Upgrade request
2. Server responds with "101 Switching Protocols"
3. Connection "upgrades" to WebSocket
4. Both sides freely send/receive messages`,
        inPalantir: "PaLANtir uses WebSocket for signaling: receiving announce messages (IP/port candidates), notifications about new/leaving peers, and peer list on connection."
      },
      {
        title: "Full Mesh Network",
        description: `Full mesh is a network topology where every node is directly connected to every other node. For N nodes, there are N*(N-1)/2 separate connections.

Example with 4 players: 4 * 3 / 2 = 6 connections

Pros:
• Optimal latency — every communication is direct
• No single point of failure
• Each pair has independent encryption

Cons:
• Connections grow quadratically — 10 players = 45 connections
• Each peer must maintain state for every other peer

For gaming (typically 2-8 players), full mesh is ideal.`,
        diagram: `  4 players — Full Mesh:

      A ══════════ B
      │╲          ╱│
      │  ╲      ╱  │
      │    ╲  ╱    │
      │     ╳      │
      │   ╱   ╲    │
      │ ╱       ╲  │
      C ══════════ D

  6 independent tunnels,
  each with their own AES keys`,
        inPalantir: "PaLANtir uses full mesh — each player pair has a separate Noise handshake, separate AES-256-GCM keys, and independent nonce counters."
      }
    ],

    "PaLANtir Specific": [
      {
        title: "How PaLANtir Works",
        description: `Complete flow from startup to playing a game:

1. REGISTRATION/LOGIN — Client sends username/password to server. Server hashes with bcrypt, returns JWT.

2. CREATE/JOIN NETWORK — Creator gets a code (e.g., 857436) and virtual IP (10.0.0.1). Other player enters code, gets 10.0.0.2.

3. TUN ADAPTER — Virtual "vlan0" adapter created with assigned IP. OS adds route for 10.0.0.0/24.

4. STUN DISCOVERY — Client contacts STUN server to learn public IP:port.

5. CANDIDATES & ANNOUNCE — Publishes local + STUN candidate via WebSocket.

6. PROBING (up to 10s) — Sends probes to all candidates in parallel. First response determines optimal path. If nothing works → relay fallback.

7. NOISE HANDSHAKE — Key exchange via NN pattern + random payload for AES key derivation.

8. TUNNEL ACTIVE — TUN packets encrypted (AES-256-GCM + explicit nonce), sent to peer via UDP (direct or relay), decrypted, written to peer's TUN. Game sees a "LAN" network.`,
        diagram: `  ┌──────┐   HTTP    ┌──────────┐   HTTP    ┌──────┐
  │  A   │◀────────▶│  Server  │◀────────▶│  B   │
  │      │   WS     │  :8080   │   WS     │      │
  │      │◀────────▶│          │◀────────▶│      │
  └──┬───┘          │  Relay   │          └──┬───┘
     │              │  :8081   │              │
     │              └──────────┘              │
     │                                       │
     │  UDP P2P (encrypted)                  │
     │◀═════════════════════════════════════▶│
     │  or UDP Relay (when P2P fails)        │
     │◀════▶ Server :8081 ◀════▶            │
     │                                       │
  ┌──┴───┐                              ┌───┴──┐
  │ TUN  │  vlan0: 10.0.0.1            │ TUN  │  vlan0: 10.0.0.2
  │ Game │  "LAN network"             │ Game │  "LAN network"
  └──────┘                              └──────┘`,
        image: "/images/palantir-saruman.gif",
        inPalantir: "This is PaLANtir's complete architecture. The server only handles signaling and relay — all gaming traffic goes P2P whenever possible."
      },
      {
        title: "Firewall Setup",
        description: `For PaLANtir and LAN games to work, you need to allow network traffic on the virtual network (10.0.0.0/24).

WINDOWS (PowerShell as administrator):

1. Allow ICMP (ping):
New-NetFirewallRule -DisplayName "Palantir ICMP" -Protocol ICMPv4 -IcmpType 8 -Direction Inbound -Action Allow

2. Allow all traffic from virtual network:
New-NetFirewallRule -DisplayName "Palantir VPN All" -Direction Inbound -RemoteAddress 10.0.0.0/24 -Action Allow

3. Remove rules when done:
Remove-NetFirewallRule -DisplayName "Palantir ICMP"
Remove-NetFirewallRule -DisplayName "Palantir VPN All"

LINUX (terminal):

1. Allow all traffic from virtual network:
sudo iptables -A INPUT -s 10.0.0.0/24 -j ACCEPT

Or with ufw:
sudo ufw allow from 10.0.0.0/24

NOTE: These rules only allow traffic from the virtual network (10.0.0.x). The only way a packet can have that source address is through PaLANtir's encrypted tunnel. Security risk is minimal.`,
        inPalantir: "Without these rules, ping works but games can't communicate. Windows specifically blocks incoming traffic on new network adapters (vlan0 is classified as 'Public' network)."
      },
      {
        title: "Known Issues and Solutions",
        description: `TUN adapter requires admin privileges:
• Windows: run .exe as Administrator
• Linux: use sudo ./palantir

Can't connect to server:
• Check if server is running
• Check if URL is correct in code
• Check if port 8080 (TCP) is open on server

Probe timeout (10 seconds):
• Both peers behind symmetric NAT
• PaLANtir automatically switches to relay
• Relay adds latency but works

Game can't find server (LAN discovery):
• Check firewall rules (see "Firewall Setup")
• Check if pings work between peers
• Some games need specific ports open

High latency through relay:
• Relay adds ~30-80ms (depends on server distance)
• Normal for symmetric NAT scenarios
• For better performance, use cone NAT networks`,
        inPalantir: "Most issues are resolved by checking the firewall and ensuring both clients run the same code version."
      }
    ]
  }
}
