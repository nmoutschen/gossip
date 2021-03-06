package gossip

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

//parseURL takes a url generated by httptest.Server and returns an Addr
func parseURL(url string) Addr {
	urlParts := strings.Split(url[7:], ":")
	ip := urlParts[0]
	port, _ := strconv.Atoi(urlParts[1])

	addr := Addr{ip, port}

	return addr
}

func TestNewPeer(t *testing.T) {
	addr := Addr{"127.0.0.1", 8080}
	p := NewPeer(addr, nil)
	if p.Addr != addr {
		t.Errorf("p.Addr == %v; want %v", p.Addr, addr)
	}
	maxLastSuccess := time.Now().Add(-p.config.Node.MaxPingDelay)
	if p.LastSuccess.Before(maxLastSuccess) {
		t.Errorf("p.LastSuccess == %v; want >= %v", p.LastSuccess, maxLastSuccess)
	}
}

func TestPeerCanPeer(t *testing.T) {
	p := NewPeer(Addr{"127.0.0.1", 8080}, nil)
	if p.CanPeer(p) != false {
		t.Errorf("p.CanPeer(p) == %t; want %t", p.CanPeer(p), false)
	}

	p2 := NewPeer(Addr{"127.0.0.1", 8081}, nil)
	if p.CanPeer(p2) != true {
		t.Errorf("p.CanPeer(p2) == %t before peering; want %t", p.CanPeer(p), true)
	}

	p.Peers = []*Peer{p2}
	if p.CanPeer(p2) != false {
		t.Errorf("p.CanPeer(p2) == %t when already peered; want %t", p.CanPeer(p), false)
	}
}

func TestPeerDelete(t *testing.T) {
	//Setup node
	var received bool
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("r.Method == %s; want %s", r.Method, "DELETE")
		}
		if r.URL.Path != "/peers" {
			t.Errorf("r.URL.PATH == %s; want %s", r.URL.Path, "/peers")
		}
		received = true
		response(w, r, http.StatusOK, "Peering request received")
	}))
	defer func() { testServer.Close() }()
	peer := NewPeer(parseURL(testServer.URL), nil)

	peer.SendPeerDeletionRequest(Addr{"127.0.0.1", 8080})

	if !received {
		t.Errorf("HTTP Server did not receive a request")
	}
}

func TestPeerGet(t *testing.T) {
	testCases := []State{
		{0, "Test Data"},
		{time.Now().UnixNano(), "Other test data"},
	}

	for _, testCase := range testCases {
		func() {
			p := &Peer{config: DefaultConfig}
			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("r.Method == %s; want %s", r.Method, "GET")
				}
				if r.URL.Path != "/" {
					t.Errorf("r.URL.PATH == %s; want %s", r.URL.Path, "/")
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(testCase)
			}))
			defer func() { testServer.Close() }()
			p.Addr = parseURL(testServer.URL)

			state, err := p.Get()

			if err != nil {
				t.Errorf("err == %v; want %v", err, nil)
			}
			if p.LastSuccess == (time.Time{}) {
				t.Errorf("p.LastSuccess == %v after p.Get()", p.LastSuccess)
			}
			if p.Attempts != 0 {
				t.Errorf("p.Attempts == %d after p.Get(); want 0", p.Attempts)
			}
			if p.LastState != testCase.Timestamp {
				t.Errorf("p.LastState == %d after p.Get(); want %d", p.LastState, testCase.Timestamp)
			}
			if state != testCase {
				t.Errorf("p.Get() == %v; want %v", state, testCase)
			}
		}()
	}
}

func TestPeerGetFail(t *testing.T) {
	p := &Peer{config: DefaultConfig}
	pState := State{
		Timestamp: time.Now().UnixNano(),
		Data:      "Test Data",
	}
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("r.Method == %s; want %s", r.Method, "GET")
		}
		if r.URL.Path != "/" {
			t.Errorf("r.URL.PATH == %s; want %s", r.URL.Path, "/")
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(pState)
	}))
	defer func() { testServer.Close() }()
	p.Addr = parseURL(testServer.URL)

	state, err := p.Get()

	if err == nil {
		t.Errorf("err == %v", err)
	}
	if p.LastSuccess != (time.Time{}) {
		t.Errorf("p.LastSuccess == %v after failed p.Get(); want %d", p.LastSuccess, 0)
	}
	if p.Attempts != 1 {
		t.Errorf("p.Attempts == %d after failed p.Get(); want %d", p.Attempts, 1)
	}
	if p.LastState != 0 {
		t.Errorf("p.LastState == %d after failed p.Get(); want %d", p.LastState, 0)
	}
	if state != (State{}) {
		t.Errorf("p.Get() == %v; want %v", state, State{})
	}
}
func TestPeerGetFail2(t *testing.T) {
	p := &Peer{config: DefaultConfig}
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("r.Method == %s; want %s", r.Method, "GET")
		}
		if r.URL.Path != "/" {
			t.Errorf("r.URL.PATH == %s; want %s", r.URL.Path, "/")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("This should not work"))
	}))
	defer func() { testServer.Close() }()
	p.Addr = parseURL(testServer.URL)

	state, err := p.Get()

	if err == nil {
		t.Errorf("err == %v", err)
	}
	if p.LastSuccess != (time.Time{}) {
		t.Errorf("p.LastSuccess == %v after failed p.Get(); want %d", p.LastSuccess, 0)
	}
	if p.Attempts != 1 {
		t.Errorf("p.Attempts == %d after failed p.Get(); want %d", p.Attempts, 1)
	}
	if p.LastState != 0 {
		t.Errorf("p.LastState == %d after failed p.Get(); want %d", p.LastState, 0)
	}
	if state != (State{}) {
		t.Errorf("p.Get() == %v; want %v", state, State{})
	}
}

func TestPeerGetPeers(t *testing.T) {
	testCases := []PeersResponse{
		{Peers: nil},
		{Peers: []Addr{Addr{"127.0.0.1", 8080}}},
		{Peers: []Addr{Addr{"127.0.0.1", 8080}, Addr{"127.0.0.1", 8081}}},
	}

	for _, testCase := range testCases {
		func() {
			p := &Peer{config: DefaultConfig}
			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("r.Method == %s; want %s", r.Method, "GET")
				}
				if r.URL.Path != "/peers" {
					t.Errorf("r.URL.PATH == %s; want %s", r.URL.Path, "/peers")
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(testCase)
			}))
			defer func() { testServer.Close() }()
			p.Addr = parseURL(testServer.URL)

			addrs, err := p.GetPeers()

			if err != nil {
				t.Errorf("err == %v; want %v", err, nil)
			}
			if p.LastSuccess == (time.Time{}) {
				t.Errorf("p.LastSuccess == %v after p.GetPeers()", p.LastSuccess)
			}
			if p.Attempts != 0 {
				t.Errorf("p.Attempts == %d after p.GetPeers(); want 0", p.Attempts)
			}
			if len(addrs) != len(testCase.Peers) {
				t.Errorf("p.GetPeers() == %v; want %v", addrs, testCase.Peers)
			}
		}()
	}
}

func TestPeerGetPeersFail(t *testing.T) {
	p := &Peer{config: DefaultConfig}
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("r.Method == %s; want %s", r.Method, "GET")
		}
		if r.URL.Path != "/peers" {
			t.Errorf("r.URL.PATH == %s; want %s", r.URL.Path, "/peers")
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(PeersResponse{
			Peers: []Addr{Addr{"127.0.0.1", 8080}},
		})
	}))
	defer func() { testServer.Close() }()
	p.Addr = parseURL(testServer.URL)

	addrs, err := p.GetPeers()

	if err == nil {
		t.Errorf("err == %v", err)
	}
	if p.LastSuccess != (time.Time{}) {
		t.Errorf("p.LastSuccess == %v after failed p.GetPeers(); want %d", p.LastSuccess, 0)
	}
	if p.Attempts != 1 {
		t.Errorf("p.Attempts == %d after failed p.GetPeers(); want %d", p.Attempts, 1)
	}
	if p.LastState != 0 {
		t.Errorf("p.LastState == %d after failed p.GetPeers(); want %d", p.LastState, 0)
	}
	if addrs != nil {
		t.Errorf("p.GetPeers() == %v; want %v", addrs, nil)
	}
}

func TestPeerGetPeersFail2(t *testing.T) {
	p := &Peer{config: DefaultConfig}
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("r.Method == %s; want %s", r.Method, "GET")
		}
		if r.URL.Path != "/peers" {
			t.Errorf("r.URL.PATH == %s; want %s", r.URL.Path, "/peers")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("This should not work"))
	}))
	defer func() { testServer.Close() }()
	p.Addr = parseURL(testServer.URL)

	addrs, err := p.GetPeers()

	if err == nil {
		t.Errorf("err == %v", err)
	}
	if p.LastSuccess != (time.Time{}) {
		t.Errorf("p.LastSuccess == %v after failed p.GetPeers(); want %d", p.LastSuccess, 0)
	}
	if p.Attempts != 1 {
		t.Errorf("p.Attempts == %d after failed p.GetPeers(); want %d", p.Attempts, 1)
	}
	if p.LastState != 0 {
		t.Errorf("p.LastState == %d after failed p.GetPeers(); want %d", p.LastState, 0)
	}
	if addrs != nil {
		t.Errorf("p.GetPeers() == %v; want %v", addrs, nil)
	}
}

func TestPeerIsIrrecoverable(t *testing.T) {
	p := &Peer{config: DefaultConfig}
	testCases := []struct {
		Expected    bool
		LastSuccess time.Time
	}{
		{false, time.Now()},
		{false, time.Now().Add(-p.config.Node.MaxPingDelay + 5*time.Second)},
		{true, time.Time{}},
		{true, time.Now().Add(-p.config.Node.MaxPingDelay - 5*time.Second)},
	}

	for _, testCase := range testCases {
		p.LastSuccess = testCase.LastSuccess
		if p.IsIrrecoverable() != testCase.Expected {
			t.Errorf("p.IsIrrecoverable() == %t with p.LastSuccess == %v; want %t", p.IsIrrecoverable(), p.LastSuccess, testCase.Expected)
		}
	}
}

func TestPeerIsCtrlIrrecoverable(t *testing.T) {
	p := &Peer{config: DefaultConfig}
	testCases := []struct {
		Expected    bool
		LastSuccess time.Time
	}{
		{false, time.Now()},
		{false, time.Now().Add(-p.config.Controller.MaxScanDelay + 5*time.Second)},
		{true, time.Time{}},
		{true, time.Now().Add(-p.config.Controller.MaxScanDelay - 5*time.Second)},
	}

	for _, testCase := range testCases {
		p.LastSuccess = testCase.LastSuccess
		if p.IsCtrlIrrecoverable() != testCase.Expected {
			t.Errorf("p.IsCtrlIrrecoverable() == %t with p.LastSuccess == %v; want %t", p.IsCtrlIrrecoverable(), p.LastSuccess, testCase.Expected)
		}
	}
}

func TestPeerIsUnreachable(t *testing.T) {
	p := &Peer{config: DefaultConfig}
	testCases := []struct {
		Expected bool
		Attempts int
	}{
		{false, 0},
		{false, DefaultConfig.Peer.MaxAttempts - 1},
		{true, DefaultConfig.Peer.MaxAttempts},
	}

	for _, testCase := range testCases {
		p.Attempts = testCase.Attempts
		if p.IsUnreachable() != testCase.Expected {
			t.Errorf("p.IsUnreachable() == %t with p.Attempts == %d; want %t", p.IsUnreachable(), p.Attempts, testCase.Expected)
		}
	}
}

func TestPeerPing(t *testing.T) {
	testCases := []struct {
		LastState int64
	}{
		{0},
		{time.Now().UnixNano()},
	}

	for _, testCase := range testCases {
		func() {
			p := &Peer{config: DefaultConfig}
			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("r.Method == %s; want %s", r.Method, "GET")
				}
				if r.URL.Path != "/status" {
					t.Errorf("r.URL.PATH == %s; want %s", r.URL.Path, "/status")
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(StatusResponse{
					LastState: testCase.LastState,
				})
			}))
			defer func() { testServer.Close() }()
			p.Addr = parseURL(testServer.URL)

			p.Ping()

			if p.LastSuccess == (time.Time{}) {
				t.Errorf("p.LastSuccess == %v after p.Ping()", p.LastSuccess)
			}
			if p.Attempts != 0 {
				t.Errorf("p.Attempts == %d after p.Ping(); want 0", p.Attempts)
			}
			if p.LastState != testCase.LastState {
				t.Errorf("p.LastState == %d after p.Ping(); want %d", p.LastState, testCase.LastState)
			}
		}()
	}
}

func TestPeerPingFail(t *testing.T) {
	p := &Peer{config: DefaultConfig}
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("r.Method == %s; want %s", r.Method, "GET")
		}
		if r.URL.Path != "/status" {
			t.Errorf("r.URL.PATH == %s; want %s", r.URL.Path, "/status")
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(StatusResponse{
			LastState: 5000,
		})
	}))
	defer func() { testServer.Close() }()
	p.Addr = parseURL(testServer.URL)

	p.Ping()

	if p.LastState != 0 {
		t.Errorf("p.LastState == %d after failed p.Ping(); want %d", p.LastState, 0)
	}
	if p.LastSuccess != (time.Time{}) {
		t.Errorf("p.LastSuccess == %v after failed p.Ping(); want %v", p.LastSuccess, time.Time{})
	}
	if p.Attempts != 1 {
		t.Errorf("p.Attempts == %d after failed p.Ping(); want %d", p.Attempts, 1)
	}
}

func TestPeerPingFail2(t *testing.T) {
	p := &Peer{config: DefaultConfig}
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("r.Method == %s; want %s", r.Method, "GET")
		}
		if r.URL.Path != "/status" {
			t.Errorf("r.URL.PATH == %s; want %s", r.URL.Path, "/status")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("This should not work"))
	}))
	defer func() { testServer.Close() }()
	p.Addr = parseURL(testServer.URL)

	p.Ping()

	if p.LastState != 0 {
		t.Errorf("p.LastState == %d after failed p.Ping(); want %d", p.LastState, 0)
	}
	if p.LastSuccess != (time.Time{}) {
		t.Errorf("p.LastSuccess == %v after failed p.Ping(); want %d", p.LastSuccess, 0)
	}
	if p.Attempts != 1 {
		t.Errorf("p.Attempts == %d after failed p.Ping(); want %d", p.Attempts, 1)
	}
}

func TestPeerSend(t *testing.T) {
	p := &Peer{config: DefaultConfig}
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("r.Method == %s; want %s", r.Method, "POST")
		}
		if r.URL.Path != "/" {
			t.Errorf("r.URL.PATH == %s; want %s", r.URL.Path, "/")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Response{
			Message: "State received",
		})
	}))
	defer func() { testServer.Close() }()
	p.Addr = parseURL(testServer.URL)
	state := State{
		Timestamp: time.Now().UnixNano(),
		Data:      "Test Data",
	}

	p.Send(state)
	if p.LastSuccess == (time.Time{}) {
		t.Errorf("p.LastSuccess == %v after p.Send()", p.LastSuccess)
	}
	if p.Attempts != 0 {
		t.Errorf("p.Attempts == %d after p.Send(); want %d", p.Attempts, 0)
	}
}

func TestPeerSendFail(t *testing.T) {
	p := &Peer{config: DefaultConfig}
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("r.Method == %s; want %s", r.Method, "POST")
		}
		if r.URL.Path != "/" {
			t.Errorf("r.URL.PATH == %s; want %s", r.URL.Path, "/")
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{
			Message: "State received",
		})
	}))
	defer func() { testServer.Close() }()
	p.Addr = parseURL(testServer.URL)
	state := State{
		Timestamp: time.Now().UnixNano(),
		Data:      "Test Data",
	}

	p.Send(state)

	if p.LastSuccess != (time.Time{}) {
		t.Errorf("p.LastSuccess == %v after failed p.Send(); want %d", p.LastSuccess, 0)
	}
	if p.Attempts != 1 {
		t.Errorf("p.Attempts == %d after failed p.Send(); want %d", p.Attempts, 1)
	}
}

func TestPeerSendUnreachable(t *testing.T) {
	p := &Peer{config: DefaultConfig}
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("r.Method == %s; want %s", r.Method, "POST")
		}
		if r.URL.Path != "/" {
			t.Errorf("r.URL.PATH == %s; want %s", r.URL.Path, "/")
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{
			Message: "State received",
		})
	}))
	defer func() { testServer.Close() }()
	p.Addr = parseURL(testServer.URL)
	p.Attempts = DefaultConfig.Peer.MaxAttempts
	state := State{
		Timestamp: time.Now().UnixNano(),
		Data:      "Test Data",
	}

	p.Send(state)

	if p.LastSuccess != (time.Time{}) {
		t.Errorf("p.LastSuccess == %v after unreachable p.Send(); want %d", p.LastSuccess, 0)
	}
	if p.Attempts != DefaultConfig.Peer.MaxAttempts {
		t.Errorf("p.Attempts == %d after unreachable p.Send(); want %d", p.Attempts, DefaultConfig.Peer.MaxAttempts)
	}
}

func TestPeerSendPeeringRequest(t *testing.T) {
	p := &Peer{config: DefaultConfig}
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("r.Method == %s; want %s", r.Method, "POST")
		}
		if r.URL.Path != "/peers" {
			t.Errorf("r.URL.PATH == %s; want %s", r.URL.Path, "/peers")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Response{
			Message: "Peering request received",
		})
	}))
	defer func() { testServer.Close() }()
	p.Addr = parseURL(testServer.URL)
	addr := Addr{"127.0.0.1", 8080}

	p.SendPeeringRequest(addr)

	if p.LastSuccess == (time.Time{}) {
		t.Errorf("p.LastSuccess == %v after p.SendPeeringRequest()", p.LastSuccess)
	}
	if p.Attempts != 0 {
		t.Errorf("p.Attempts == %d after p.SendPeeringRequest(); want %d", p.Attempts, 0)
	}
}

func TestPeerSendPeeringRequestFail(t *testing.T) {
	p := &Peer{config: DefaultConfig}
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("r.Method == %s; want %s", r.Method, "POST")
		}
		if r.URL.Path != "/peers" {
			t.Errorf("r.URL.PATH == %s; want %s", r.URL.Path, "/peers")
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{
			Message: "Peering request received",
		})
	}))
	defer func() { testServer.Close() }()
	p.Addr = parseURL(testServer.URL)
	addr := Addr{"127.0.0.1", 8080}

	p.SendPeeringRequest(addr)

	if p.LastSuccess != (time.Time{}) {
		t.Errorf("p.LastSuccess == %v after failed p.SendPeeringRequest(); want %d", p.LastSuccess, 0)
	}
	if p.Attempts != 1 {
		t.Errorf("p.Attempts == %d after failed p.SendPeeringRequest(); want %d", p.Attempts, 1)
	}
}

func TestPeerUpdateStatus(t *testing.T) {
	p := &Peer{config: DefaultConfig}

	p.UpdateStatus(false)
	if p.LastSuccess != (time.Time{}) {
		t.Errorf("p.LastSuccess == %v after p.UpdateStatus(); want %d", p.LastSuccess, 0)
	}
	if p.Attempts != 1 {
		t.Errorf("p.Attempts == %d after p.UpdateStatus(); want %d", p.Attempts, 1)
	}

	p = &Peer{config: DefaultConfig}
	p.UpdateStatus(true)
	if p.LastSuccess == (time.Time{}) {
		t.Errorf("p.LastSuccess == %v after p.UpdateStatus()", p.LastSuccess)
	}
	if p.Attempts != 0 {
		t.Errorf("p.Attempts == %d after p.UpdateStatus(); want %d", p.Attempts, 0)
	}
}
