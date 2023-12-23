package main

import (
        "bytes"
        "encoding/json"
        "flag"
        "fmt"
        "net/http"
        "os"
        "sync"
        "time"

        "github.com/pion/webrtc/v4"
)

type AnswerResut struct {

  Answer webrtc.SessionDescription `json:"answer"`

  Ices []webrtc.ICECandidate   `json:"ices"`
}

func main() { //nolint:gocognit
        answerAddr := flag.String("answer-address", "127.0.0.1:60000", "Address that the Answer HTTP server is hosted on.")
        flag.Parse()

        var candidatesMux sync.Mutex
        pendingCandidates := make([]*webrtc.ICECandidate, 0)

        // Everything below is the Pion WebRTC API! Thanks for using it ❤️.

        // Prepare the configuration
        config := webrtc.Configuration{
                ICEServers: []webrtc.ICEServer{
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
                if cErr := peerConnection.Close(); cErr != nil {
                        fmt.Printf("cannot close peerConnection: %v\n", cErr)
                }
        }()

        // When an ICE candidate is available send to the other Pion instance
        // the other Pion instance will add this candidate by calling AddICECandidate
        peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
                if c == nil {
                        return
                }

                candidatesMux.Lock()
                defer candidatesMux.Unlock()

                fmt.Printf("ice candidate %v \n", *c)
                pendingCandidates = append(pendingCandidates, c)
        })

        // Start HTTP server that accepts requests from the answer process
        // nolint: gosec
        go func() { panic(http.ListenAndServe(":50000", nil)) }()

        // Create a datachannel with label 'data'
        dataChannel, err := peerConnection.CreateDataChannel("data", nil)
        if err != nil {
                panic(err)
        }

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

        // Register channel opening handling
        dataChannel.OnOpen(func() {
                fmt.Printf("Data channel '%s'-'%d' open. Random messages will now be sent to any connected DataChannels every 5 seconds\n", dataChannel.Label(), dataChannel.ID())

                for range time.NewTicker(5 * time.Second).C {
                        message := "from offer"
                        fmt.Printf("Sending '%s'\n", message)

                        // Send the message as text
                        sendTextErr := dataChannel.SendText(message)
                        if sendTextErr != nil {
                                panic(sendTextErr)
                        }
                }
        })

        // Register text message handling
        dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
                fmt.Printf("Message from DataChannel '%s': '%s'\n", dataChannel.Label(), string(msg.Data))
        })


        // Create an offer to send to the other process
        offer, err := peerConnection.CreateOffer(nil)
        if err != nil {
                panic(err)
        }

        // Sets the LocalDescription, and starts our UDP listeners
        // Note: this will start the gathering of ICE candidates
        if err = peerConnection.SetLocalDescription(offer); err != nil {
                panic(err)
        }

        time.Sleep(2 * time.Second)

        offer, err = peerConnection.CreateOffer(nil)

        // Send our offer to the HTTP server listening in the other process
        payload, err := json.Marshal(offer)
        if err != nil {
                panic(err)
        }
        resp, err := http.Post(fmt.Sprintf("http://%s/sdp", *answerAddr), "application/json; charset=utf-8", bytes.NewReader(payload)) // nolint:noctx
        if err != nil {
                panic(err)
        }

        result := new(AnswerResut)
        json.NewDecoder(resp.Body).Decode(&result)

        answer := result.Answer
        ides := result.Ices

        peerConnection.SetRemoteDescription(answer)

        for _, c := range ides {
           fmt.Printf("answer candidate %v \n", c);
           c_bytes, _ := json.Marshal(c)
           peerConnection.AddICECandidate(webrtc.ICECandidateInit{Candidate: string(c_bytes)})
        }

        // Block forever
        select {}
}
