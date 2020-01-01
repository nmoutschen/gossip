package gossip

//CtrlPeerResponse is a single node as part of a CtrlPeersResponse struct.
type CtrlPeerResponse struct {
	Addr  Addr   `json:"addr"`
	Peers []Addr `json:"peers"`
}

//CtrlPeersResponse is the response sent for a /nodes request to a controller.
type CtrlPeersResponse struct {
	Nodes []CtrlPeerResponse `json:"nodes"`
}

//PeersResponse is the response sent for a /peers request.
type PeersResponse struct {
	Peers []Addr `json:"peers"`
}

//Response is the response sent to requests when an error occurs.
type Response struct {
	Message string `json:"message"`
}

/*StatusResponse is the response sent for a /status request.

This contains the timestamp for the latest known state.
*/
type StatusResponse struct {
	LastState int64 `json:"lastState"`
}
