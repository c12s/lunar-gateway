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

func (server *LunarServer) setupNamespaces() {
	secrets := server.r.PathPrefix("/namespaces").Subrouter()
	secrets.HandleFunc("/list", auth(server.rightsList(server.listNamespaces()))).Methods("GET")
	secrets.HandleFunc("/mutate", auth(server.rightsMutate(server.mutateNamespaces()))).Methods("POST")
}

var naq = [...]string{"user", "labels", "compare", "name"}

func (s *LunarServer) listNamespaces() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		span, _ := sg.FromRequest(r, "listNamespaces")
		defer span.Finish()
		fmt.Println(span)

		req := listToProto(r.URL.Query(), cPb.ReqKind_NAMESPACES)
		client := NewMeridianClient(s.clients[MERIDIAN])
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

		rresp := protoToNSListResp(resp)
		data, rerr := json.Marshal(rresp)
		if rerr != nil {
			span.AddLog(&sg.KV{"proto to json error", rerr.Error()})
			sendErrorMessage(w, rerr.Error(), http.StatusBadRequest)
			return
		}
		sendJSONResponse(w, string(data))
	}
}

func (s *LunarServer) mutateNamespaces() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		span, _ := sg.FromRequest(r, "mutateNamespaces")
		defer span.Finish()
		fmt.Println(span)

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			span.AddLog(&sg.KV{"Failed to read the request body", err.Error()})
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		data := &model.NMutateRequest{}
		if err := json.Unmarshal(body, data); err != nil {
			span.AddLog(&sg.KV{"Could not decode the request body as JSON", err.Error()})
			sendErrorMessage(w, "Could not decode the request body as JSON", http.StatusBadRequest)
			return
		}

		req := mutateNSToProto(data)
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
