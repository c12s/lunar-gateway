package server

import (
	"context"
	"encoding/json"
	"fmt"
	sg "github.com/c12s/stellar-go"
	"net/http"
	"time"
)

func (server *LunarServer) setupNodes() {
	nodes := server.r.PathPrefix("/nodes").Subrouter()
	nodes.HandleFunc("/list", auth(server.rightsList(server.listNodes()))).Methods("GET")
}

func (s *LunarServer) listNodes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		span, _ := sg.FromRequest(r, "listNodes")
		defer span.Finish()
		fmt.Println(span)
		fmt.Println(span.Serialize())

		req := listNodesToProto(r.URL.Query())
		client := NewMagnetarClient(s.clients[MAGNETAR])
		ctx, cancel := context.WithTimeout(
			appendToken(
				sg.NewTracedGRPCContext(nil, span),
				r.Header["Authorization"][0],
			),
			10*time.Second,
		)
		defer cancel()

		resp, err := client.Query(ctx, req)
		if err != nil {
			span.AddLog(&sg.KV{"celestial.list error", err.Error()})
			sendErrorMessage(w, err.Error(), http.StatusBadRequest)
			return
		}

		rresp := protoToNodesListResp(resp)
		data, rerr := json.Marshal(rresp)
		if rerr != nil {
			span.AddLog(&sg.KV{"proto to json error", rerr.Error()})
			sendErrorMessage(w, rerr.Error(), http.StatusBadRequest)
			return
		}
		sendJSONResponse(w, string(data))
	}
}
