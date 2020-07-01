package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/c12s/lunar-gateway/model"
	aPb "github.com/c12s/scheme/apollo"
	stellar "github.com/c12s/stellar-go"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func auth(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		trace := stellar.Init("lunar-gateway")
		defer trace.Finish()

		span := trace.Span("auth")
		defer span.Finish()
		fmt.Println(span)
		span.AddTag(
			&stellar.KV{"app", "test"},
			&stellar.KV{"trace", "mytrace"},
		)

		if _, ok := r.Header["Authorization"]; !ok {
			span.AddLog(&stellar.KV{"auth header error", "missing authorization token"})
			sendErrorMessage(w, "missing authorization token", http.StatusBadRequest)
			return
		}

		if _, ok := r.URL.Query()["user"]; !ok {
			span.AddLog(&stellar.KV{"auth user error", "missing user id"})
			sendErrorMessage(w, "missing user id", http.StatusBadRequest)
			return
		}
		next(w, stellar.TracedRequest(r, span))
	})
}

func (server *LunarServer) rightsList(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span, _ := stellar.FromRequest(r, "rightsList")
		defer span.Finish()
		fmt.Println(span)

		if len(r.URL.Query()) == 0 {
			span.AddLog(&stellar.KV{"query error", "missing query parameters"})
			sendErrorMessage(w, "missing query parameters", http.StatusBadRequest)
			return
		}
		extras := map[string]*aPb.OptExtras{}
		for k, v := range r.URL.Query() {
			extras[k] = &aPb.OptExtras{Data: v}
		}

		spl := strings.Split(r.URL.Path, "/")
		req := &aPb.AuthOpt{
			Data: map[string]string{
				"intent": "auth",
				"action": spl[4],
				"kind":   spl[3],
				"token":  r.Header["Authorization"][0],
			},
			Extras: extras,
		}

		client := NewApolloClient(server.clients[APOLLO])
		ctx, cancel := context.WithTimeout(
			appendToken(
				stellar.NewTracedGRPCContext(nil, span),
				r.Header["Authorization"][0],
			), 10*time.Second)
		defer cancel()

		resp, err := client.Auth(ctx, req)
		if err != nil {
			span.AddLog(&stellar.KV{"apollo.auth error", err.Error()})
			sendErrorMessage(w, err.Error(), http.StatusBadRequest)
			return
		}

		if !resp.Value {
			span.AddLog(&stellar.KV{"apollo.auth value", resp.Data["message"]})
			sendErrorMessage(w, resp.Data["message"], http.StatusBadRequest)
			return
		}
		next(w, r)
	})
}

func (server *LunarServer) rightsMutate(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span, _ := stellar.FromRequest(r, "rightsMutate")
		defer span.Finish()
		fmt.Println(span)

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			span.AddLog(&stellar.KV{"Failed to read the request body", err.Error()})
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		r.Body = ioutil.NopCloser(bytes.NewBuffer(body)) //ReadAll remove contet, so save it!

		spl := strings.Split(r.URL.Path, "/")
		extras := map[string]*aPb.OptExtras{}
		req := &aPb.AuthOpt{}

		if spl[3] == "topology" {
			data := &model.TMutateRequest{}
			if err := json.Unmarshal(body, data); err != nil {
				span.AddLog(&stellar.KV{"Could not decode the request body as JSON", err.Error()})
				sendErrorMessage(w, "Could not decode the request body as JSON", http.StatusBadRequest)
				return
			}

			l := []string{}
			for lk, lv := range data.Payload.Labels {
				l = append(l, strings.Join([]string{lk, lv}, ":"))
			}
			extras["labels"] = &aPb.OptExtras{Data: l}
			extras["name"] = &aPb.OptExtras{Data: []string{data.Payload.Name}}

			req.Data = map[string]string{
				"intent":       "auth",
				"action":       spl[4],
				"kind":         spl[3],
				"user":         r.URL.Query()["user"][0],
				"token":        r.Header["Authorization"][0],
				"namespace":    data.MTData.Namespace,
				"queue":        data.MTData.Queue,
				"forceNSQueue": strconv.FormatBool(data.MTData.ForceNSQueue),
			}
			req.Extras = extras

		} else if spl[3] == "roles" {
			data := &model.RMutateRequest{}
			if err := json.Unmarshal(body, data); err != nil {
				span.AddLog(&stellar.KV{"Could not decode the request body as JSON", err.Error()})
				sendErrorMessage(w, "Could not decode the request body as JSON", http.StatusBadRequest)
				return
			}

			extras["user"] = &aPb.OptExtras{Data: []string{data.Payload.User}}
			extras["resources"] = &aPb.OptExtras{Data: data.Payload.Resources}
			extras["verbs"] = &aPb.OptExtras{Data: data.Payload.Verbs}

			req.Data = map[string]string{
				"intent":       "auth",
				"action":       spl[4],
				"kind":         spl[3],
				"user":         r.URL.Query()["user"][0],
				"token":        r.Header["Authorization"][0],
				"namespace":    data.MTData.Namespace,
				"queue":        data.MTData.Queue,
				"forceNSQueue": strconv.FormatBool(data.MTData.ForceNSQueue),
			}
			req.Extras = extras

		} else if spl[3] != "namespaces" {
			data := &model.MutateRequest{}
			if err := json.Unmarshal(body, data); err != nil {
				span.AddLog(&stellar.KV{"Could not decode the request body as JSON", err.Error()})
				sendErrorMessage(w, "Could not decode the request body as JSON", http.StatusBadRequest)
				return
			}

			for _, r := range data.Regions {
				tmp := []string{}
				for _, c := range r.Clusters {
					tmp = append(tmp, c.ID)

					l := []string{}
					for lk, lv := range c.Selector.Labels {
						l = append(l, strings.Join([]string{lk, lv}, ":"))
					}
					tmp = append(tmp, strings.Join(l, ","))

					cmp := []string{}
					for ck, cv := range c.Selector.Compare {
						cmp = append(cmp, strings.Join([]string{ck, cv}, ":"))
					}
					tmp = append(tmp, strings.Join(cmp, ","))
				}
				extras[r.ID] = &aPb.OptExtras{Data: tmp}
			}

			req.Data = map[string]string{
				"intent":       "auth",
				"action":       spl[4],
				"kind":         spl[3],
				"user":         r.URL.Query()["user"][0],
				"token":        r.Header["Authorization"][0],
				"namespace":    data.MTData.Namespace,
				"queue":        data.MTData.Queue,
				"forceNSQueue": strconv.FormatBool(data.MTData.ForceNSQueue),
			}
			req.Extras = extras
		} else {
			data := &model.NMutateRequest{}
			if err := json.Unmarshal(body, data); err != nil {
				span.AddLog(&stellar.KV{"Could not decode the request body as JSON", err.Error()})
				sendErrorMessage(w, "Could not decode the request body as JSON", http.StatusBadRequest)
				return
			}

			l := []string{}
			for lk, lv := range data.Labels {
				l = append(l, strings.Join([]string{lk, lv}, ":"))
			}
			extras["labels"] = &aPb.OptExtras{Data: l}
			extras["namespace"] = &aPb.OptExtras{Data: []string{data.Name}}

			req.Data = map[string]string{
				"intent":       "auth",
				"action":       spl[4],
				"kind":         spl[3],
				"user":         r.URL.Query()["user"][0],
				"token":        r.Header["Authorization"][0],
				"namespace":    data.MTData.Namespace,
				"queue":        data.MTData.Queue,
				"forceNSQueue": strconv.FormatBool(data.MTData.ForceNSQueue),
			}
			req.Extras = extras
		}

		client := NewApolloClient(server.clients[APOLLO])
		ctx, cancel := context.WithTimeout(
			appendToken(
				stellar.NewTracedGRPCContext(nil, span),
				r.Header["Authorization"][0],
			), 10*time.Second)
		defer cancel()

		resp, err := client.Auth(ctx, req)
		if err != nil {
			span.AddLog(&stellar.KV{"apollo.auth error", err.Error()})
			sendErrorMessage(w, err.Error(), http.StatusBadRequest)
			return
		}

		if !resp.Value {
			span.AddLog(&stellar.KV{"apollo.auth value", resp.Data["message"]})
			sendErrorMessage(w, resp.Data["message"], http.StatusBadRequest)
			return
		}
		next(w, r)
	})
}
