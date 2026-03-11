/*
Test Description:

The purpose of this test is to imitate the simple-wallet's strategy
for handling sensitive methods, like QueryKey; and the current
strategy assumes that the user understands:

> IF a connecting application has asked to connect WITH permissions,
> ALL requested permissions are either Authorized or Rejected by the user;
> the connection then has the authority to execute based on initial request.

If the application DID NOT ask on initial connection,
then the default for QueryKey is AlwaysDeny

If the application DID ask for permission on initial connection,
and the user Accepted the Permissions list of the request,
the connection will have permission to operate based on
that initial authorization exchange with that user until that
connection has disconnected and needs to be re-established again.
*/
package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/creachadair/jrpc2"
	"github.com/deroproject/derohe/walletapi"
	"github.com/deroproject/derohe/walletapi/xswd"
	"github.com/gorilla/websocket"
)

func TestMethods(t *testing.T) {
	// preliminaries

	// Create XSWD server
	wd, err := walletapi.Create_Encrypted_Wallet_Random("tmp.db", "temporary")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(wd.GetAddress())
	forceAsk := false
	noStore := []string{""}
	appHandler := func(ad *xswd.ApplicationData) bool { return true } // for testing purposes
	requestHandler := func(ad *xswd.ApplicationData, r *jrpc2.Request) xswd.Permission {
		if r.HasParams() {
			fmt.Println(r.Method())
			switch r.Method() {
			case "QueryKey":

				// not implemented
				return xswd.AlwaysDeny

			default:

			}
		}

		// now wait for the choice
		// if <-choice { // if accepted...
		return xswd.Allow
		// }

		// default is to deny
		// return xswd.Deny
	}

	server := xswd.NewXSWDServerWithPort(44326, wd, forceAsk, noStore, appHandler, requestHandler)
	defer server.Stop()
	time.Sleep(1 * time.Second) // wait for the server?
	if !server.IsRunning() {
		t.Log("xswd server is not running")
		t.FailNow()
	}
	// example application
	var appID = "6df99f80bc8b17340c21fa9c7613e9837cf641b1a1168433e8343337c752073c"
	var appSig = `-----BEGIN DERO SIGNED MESSAGE-----
Address: dero1qyc96tgvz8fz623snpfwjgdhlqznamcsuh8rahrh2yvsf2gqqxdljqg9a9kka
C: d30f486cc66f6d6571112fcb3aacba4f076aba439e9bd0e84bef94b06e5c851
S: 2d839f4432e1c7a2da391dd01ed9efec64831b2bbc99a47ab4a04b283005080a

NmRmOTlmODBiYzhiMTczNDBjMjFmYTljNzYxM2U5ODM3Y2Y2NDFiMWExMTY4NDMz
ZTgzNDMzMzdjNzUyMDczYw==
-----END DERO SIGNED MESSAGE-----`
	var Xswd_conn *websocket.Conn
	var AppData = xswd.ApplicationData{
		Id:          appID,
		Signature:   []byte(appSig),
		Name:        "simple-tela-deploymnet-manager",
		Description: "Creating deployments on must be simple and fun! :)",
		Url:         "http://localhost:8080",
		// Permissions: map[string]xswd.Permission{"QueryKey": xswd.AlwaysAllow},
	}
	var websocket_endpoint string = "ws://127.0.0.1:44326/xswd"

	for _, each := range os.Args {
		if !strings.Contains(each, "--ws-address=") {
			continue
		}
		endpoint := strings.Split(each, "=")[1]
		websocket_endpoint = "ws://" + endpoint + "/xswd"
	}

	fmt.Printf("Connecting to %s\n", websocket_endpoint)

	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // allow self-signed certs
	}

	conn, _, err := dialer.Dial(websocket_endpoint, nil)
	Xswd_conn = conn

	if err != nil {
		t.Error(err)
		return
	}
	postBytes := func(b []byte) []byte {
		Xswd_conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		err := Xswd_conn.WriteMessage(websocket.TextMessage, b)
		if err != nil {
			panic(err)
		}

		_, msg, err := Xswd_conn.ReadMessage()
		if err != nil {
			panic(err)
		}
		return msg
	}

	querykey := func() []byte {

		// fmt.Println(estimate)
		payload := map[string]any{
			"jsonrpc": "2.0",
			"id":      "QueryKey",
			"method":  "QueryKey",
			"params":  map[string]any{"key_type": "mnemonic"},
		}
		jsonBytes, err := json.Marshal(payload)
		if err != nil {
			return nil
		}

		var r xswd.RPCResponse
		if err := json.Unmarshal(postBytes(jsonBytes), &r); err != nil {
			return nil
		}

		fmt.Printf("%+v\n", r)
		if r.Error.(map[string]any)["code"].(float64) != -32044 {
			t.Fatal("this should only be an error")
		}

		if r.Error.(map[string]any)["message"].(string) != `Permission not granted for method "QueryKey"` {
			t.Fatal("this should only be an error")
		}

		if r.Result == nil {
			return nil
		}

		return []byte("fail")

	}

	condition := "fail"

	if server.IsRunning() {

		fmt.Println("WebSocket xswd_connected")
		if err := Xswd_conn.WriteJSON(AppData); err != nil {
			t.Error(err)
			return
		}

		fmt.Println("Auth handshake sent")
		Xswd_conn.SetReadDeadline(time.Now().Add(5 * time.Second))

		_, msg, err := Xswd_conn.ReadMessage()
		if err != nil {
			t.Fatal(err)
		}

		var res xswd.AuthorizationResponse
		if err := json.Unmarshal(msg, &res); err != nil {
			t.Fatal(err)
		}

		if !res.Accepted {
			t.Fatal("authorization rejected")
		}
	}

	if condition == string(querykey()) {
		t.FailNow()
	}
}
