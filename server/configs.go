package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/c12s/lunar-gateway/model"
	cPb "github.com/c12s/scheme/celestial"
	sg "github.com/c12s/stellar-go"
	"io/ioutil"
	"net/http"
	"time"
)

func (server *LunarServer) setupConfigs() {
	configs := server.r.PathPrefix("/configs").Subrouter()
	configs.HandleFunc("/list", auth(server.rightsList(server.listConfigs()))).Methods("GET")
	configs.HandleFunc("/mutate", auth(server.rightsMutate(server.mutateConfigs()))).Methods("POST")
}

var caq = [...]string{"labels", "compare", "user"}

func (s *LunarServer) listConfigs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		span, _ := sg.FromRequest(r, "listConfigs")
		defer span.Finish()
		fmt.Println(span)
		fmt.Println(span.Serialize())

		req := listToProto(r.URL.Query(), cPb.ReqKind_CONFIGS)
		client := NewCelestialClient(s.clients[CELESTIAL])
		ctx, cancel := context.WithTimeout(
			appendToken(
				sg.NewTracedGRPCContext(nil, span),
				r.Header["Authorization"][0],
			),
			10*time.Second,
		)
		defer cancel()

		resp, err := client.List(ctx, req)
		if err != nil {
			span.AddLog(&sg.KV{"celestial.list error", err.Error()})
			sendErrorMessage(w, err.Error(), http.StatusBadRequest)
			return
		}

		rresp := protoToConfigListResp(resp)
		data, rerr := json.Marshal(rresp)
		if rerr != nil {
			span.AddLog(&sg.KV{"proto to json error", rerr.Error()})
			sendErrorMessage(w, rerr.Error(), http.StatusBadRequest)
			return
		}
		sendJSONResponse(w, string(data))
	}
}

func (s *LunarServer) mutateConfigs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		span, _ := sg.FromRequest(r, "mutateConfigs")
		defer span.Finish()
		fmt.Println(span)
		fmt.Println("SERIALIZE ", span.Serialize())

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			span.AddLog(&sg.KV{"Failed to read the request body", err.Error()})
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		data := &model.MutateRequest{}
		if err := json.Unmarshal(body, data); err != nil {
			span.AddLog(&sg.KV{"Could not decode the request body as JSON", err.Error()})
			sendErrorMessage(w, "Could not decode the request body as JSON", http.StatusBadRequest)
			return
		}

		req := mutateToProto(data)
		client := NewBlackHoleClient(s.clients[BLACKHOLE])
		ctx, cancel := context.WithTimeout(
			appendToken(
				sg.NewTracedGRPCContext(nil, span),
				r.Header["Authorization"][0],
			),
			10*time.Second,
		)
		defer cancel()

		resp, err := client.Put(ctx, req)
		if err != nil {
			span.AddLog(&sg.KV{"blackhole.put error", err.Error()})
			sendErrorMessage(w, err.Error(), http.StatusBadRequest)
			return
		}
		sendJSONResponse(w, map[string]string{"message": resp.Msg})
	}
}
