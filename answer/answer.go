package main

import (
        "encoding/json"
        "flag"
        "fmt"
        "io"
        "net/http"
        "os"
        "sync"
        "time"

        "github.com/pion/webrtc/v4"
)

func main() { // nolint:gocognit
        flag.Parse()

        var candidatesMux sync.Mutex
        pendingCandidates := make([]*webrtc.ICECandidate, 0)
        // Everything below is the Pion WebRTC API! Thanks for using it ❤️.

        // Prepare the configuration
        config := webrtc.Configuration{
                ICEServers: []webrtc.ICEServer{
                        {
                                URLs: []string{"stun:stun.l.google.com:19302"},
                        },
                        {
                                URLs: []string{"turn:47.251.71.83:3478"},
                                Username: "user-test",
                                Credential: "123456",
                                CredentialType: webrtc.ICECredentialTypePassword,
                        },
                },
                ICETransportPolicy: webrtc.ICETransportPolicyRelay,
                BundlePolicy:       webrtc.BundlePolicyBalanced,
                RTCPMuxPolicy:      webrtc.RTCPMuxPolicyRequire,
        }

        // Create a new RTCPeerConnection
        peerConnection, err := webrtc.NewPeerConnection(config)
        if err != nil {
                panic(err)
        }
        defer func() {
                if err := peerConnection.Close(); err != nil {
                        fmt.Printf("cannot close peerConnection: %v\n", err)
                }
        }()

        // When an ICE candidate is available send to the other Pion instance
        // the other Pion instance will add this candidate by calling AddICECandidate
        peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
                if c == nil {
                        return
                }

                fmt.Printf("ice candidate %v \n", *c);

                candidatesMux.Lock()
                defer candidatesMux.Unlock()

                desc := peerConnection.RemoteDescription()
                if desc != nil {
                        pendingCandidates = append(pendingCandidates, c)
                }
        })

        // A HTTP handler that processes a SessionDescription given to us from the other Pion process
        http.HandleFunc("/sdp", func(w http.ResponseWriter, r *http.Request) {
                sdp := webrtc.SessionDescription{}
                if err := json.NewDecoder(r.Body).Decode(&sdp); err != nil {
                        panic(err)
                }

                fmt.Printf("received offer %v\n", sdp)

                if err := peerConnection.SetRemoteDescription(sdp); err != nil {
                        panic(err)
                }


                // Create an answer to send to the other process
                answer, err := peerConnection.CreateAnswer(nil)
                if err != nil {
                        panic(err)
                }

                peerConnection.SetLocalDescription(answer)

                time.Sleep(10 * time.Second)

                result := make(map[string]interface{})

                result["answer"] = answer
                result["ices"] = pendingCandidates

                fmt.Printf("return answer with ices %v\n", result)

                // 将response数据对象编码为 JSON 格式
                resDatas, _ := json.Marshal(result)

                // 返回给浏览器
                io.WriteString(w, string(resDatas))

        })

        // Set the handler for Peer connection state
        // This will notify you when the peer has connected/disconnected
        peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
                fmt.Printf("Peer Connection State has changed: %s\n", s.String())

                if s == webrtc.PeerConnectionStateFailed {
                        // Wait until PeerConnection has had no network activity for 30 seconds or another failure. It may be reconnected using an ICE Restart.
                        // Use webrtc.PeerConnectionStateDisconnected if you are interested in detecting faster timeout.
                        // Note that the PeerConnection may come back from PeerConnectionStateDisconnected.
                        // fmt.Println("Peer Connection has gone to failed exiting")
                        // os.Exit(0)
                }

                if s == webrtc.PeerConnectionStateClosed {
                        // PeerConnection was explicitly closed. This usually happens from a DTLS CloseNotify
                        fmt.Println("Peer Connection has gone to closed exiting")
                        os.Exit(0)
                }
        })

        // Register data channel creation handling
        peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
                fmt.Printf("New DataChannel %s %d\n", d.Label(), d.ID())

                // Register channel opening handling
                d.OnOpen(func() {
                        fmt.Printf("Data channel '%s'-'%d' open. Random messages will now be sent to any connected DataChannels every 5 seconds\n", d.Label(), d.ID())

                        for range time.NewTicker(5 * time.Second).C {
                                message := "from answer server"
                                fmt.Printf("Sending '%s'\n", message)

                                // Send the message as text
                                sendTextErr := d.SendText(message)
                                if sendTextErr != nil {
                                        panic(sendTextErr)
                                }
                        }
                })

                // Register text message handling
                d.OnMessage(func(msg webrtc.DataChannelMessage) {
                        fmt.Printf("Message from DataChannel '%s': '%s'\n", d.Label(), string(msg.Data))
                })
        })

        // Start HTTP server that accepts requests from the offer process to exchange SDP and Candidates
        // nolint: gosec
        panic(http.ListenAndServe(":60000", nil))
}
