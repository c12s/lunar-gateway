package server

import (
	"encoding/json"
	"github.com/c12s/lunar-gateway/model"
	bPb "github.com/c12s/scheme/blackhole"
	cPb "github.com/c12s/scheme/celestial"
	mPb "github.com/c12s/scheme/magnetar"
	sPb "github.com/c12s/scheme/stellar"
	// "io"
	"context"
	"google.golang.org/grpc/metadata"
	"log"
	"net/http"
	"sort"
	"strings"
)

func appendToken(ctx context.Context, token string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "c12stoken", token)
}

func merge(m1, m2 map[string]string) {
	for k, v := range m2 {
		m1[k] = v
	}
}

func sendJSONResponse(w http.ResponseWriter, data interface{}) {
	body, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to encode a JSON response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(body)
	if err != nil {
		log.Printf("Failed to write the response body: %v", err)
		return
	}
}

func sendJSONResponseWithHeader(w http.ResponseWriter, data interface{}, headers map[string]string) {
	body, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to encode a JSON response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	for k, v := range headers {
		w.Header().Set(k, v)
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write(body)
	if err != nil {
		log.Printf("Failed to write the response body: %v", err)
		return
	}
}

func sendErrorMessage(w http.ResponseWriter, msg string, status int) {
	body, err := json.Marshal(map[string]string{"message": msg})
	if err != nil {
		log.Printf("Failed to encode a JSON response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	w.WriteHeader(status)
	_, err = w.Write(body)
	if err != nil {
		log.Printf("Failed to write the response body: %v", err)
		return
	}
}

func cKind(kind string) bPb.CompareKind {
	switch kind {
	case all:
		return bPb.CompareKind_ALL
	case any:
		return bPb.CompareKind_ANY
	default:
		return -1
	}
}

func pKind(kind string) bPb.PayloadKind {
	switch kind {
	case file:
		return bPb.PayloadKind_FILE
	case env:
		return bPb.PayloadKind_ENV
	case action:
		return bPb.PayloadKind_ACTION
	default:
		return -1
	}
}

func sKind(kind string) bPb.StrategyKind {
	switch kind {
	case at_once:
		return bPb.StrategyKind_AT_ONCE
	case rolling_update:
		return bPb.StrategyKind_ROLLING_UPDATE
	case canary:
		return bPb.StrategyKind_CANARY
	default:
		return -1
	}
}

func tKind(kind string) bPb.TaskKind {
	switch kind {
	case Configs:
		return bPb.TaskKind_CONFIGS
	case Secrets:
		return bPb.TaskKind_SECRETS
	case Actions:
		return bPb.TaskKind_ACTIONS
	case Namespaces:
		return bPb.TaskKind_NAMESPACES
	case Roles:
		return bPb.TaskKind_ROLES
	case Topology:
		return bPb.TaskKind_TOPOLOGY
	default:
		return -1
	}
}

func mutateToProto(data *model.MutateRequest) *bPb.PutReq {
	tasks := []*bPb.PutTask{}
	for _, region := range data.Regions {
		for _, cluster := range region.Clusters {
			labels := map[string]string{}
			for k, v := range cluster.Selector.Labels {
				labels[k] = v
			}

			payload := []*bPb.Payload{}
			for _, entry := range cluster.Payload {
				values := map[string]string{}
				for k, v := range entry.Content {
					values[k] = v
				}

				pld := &bPb.Payload{
					Kind:  pKind(entry.Kind),
					Value: values,
					Index: entry.Index,
				}
				payload = append(payload, pld)
			}

			task := &bPb.PutTask{
				RegionId:  region.ID,
				ClusterId: cluster.ID,
				Strategy: &bPb.Strategy{
					Type:     sKind(cluster.Strategy.Type),
					Kind:     cluster.Strategy.Kind,
					Interval: cluster.Strategy.Interval,
				},
				Selector: &bPb.Selector{
					Kind:   cKind(cluster.Selector.Compare[kind]),
					Labels: labels,
				},
				Payload: payload,
			}
			tasks = append(tasks, task)
		}
	}

	return &bPb.PutReq{
		Version: data.Version,
		UserId:  data.Request,
		Kind:    tKind(data.Kind),
		Mtdata: &bPb.Metadata{
			TaskName:            data.MTData.TaskName,
			Timestamp:           data.MTData.Timestamp,
			Namespace:           data.MTData.Namespace,
			ForceNamespaceQueue: data.MTData.ForceNSQueue,
			Queue:               data.MTData.Queue,
		},
		Tasks: tasks,
	}
}

func mutateNSToProto(data *model.NMutateRequest) *bPb.PutReq {
	extras := map[string]string{}
	labels := []string{}
	for k, v := range data.Labels {
		pair := strings.Join([]string{k, v}, ":")
		labels = append(labels, pair)
	}

	// Add namespace labels
	sort.Strings(labels)
	extras[labels_key] = strings.Join(labels, ",")

	// Add namespace name to the extras
	extras[ns_key] = data.Name
	return &bPb.PutReq{
		Version: data.Version,
		UserId:  data.Request,
		Kind:    tKind(data.Kind),
		Mtdata: &bPb.Metadata{
			TaskName:            data.MTData.TaskName,
			Timestamp:           data.MTData.Timestamp,
			Namespace:           data.MTData.Namespace,
			ForceNamespaceQueue: data.MTData.ForceNSQueue,
			Queue:               data.MTData.Queue,
		},
		Extras: extras,
	}
}

func rolesToProto(data *model.RMutateRequest) *bPb.PutReq {
	extras := map[string]string{}
	extras["user"] = data.Payload.User
	extras["resources"] = strings.Join(data.Payload.Resources, ",")
	extras["verbs"] = strings.Join(data.Payload.Verbs, ",")
	return &bPb.PutReq{
		Version: data.Version,
		UserId:  data.Request,
		Kind:    tKind(data.Kind),
		Mtdata: &bPb.Metadata{
			TaskName:            data.MTData.TaskName,
			Timestamp:           data.MTData.Timestamp,
			Namespace:           data.MTData.Namespace,
			ForceNamespaceQueue: data.MTData.ForceNSQueue,
			Queue:               data.MTData.Queue,
		},
		Extras: extras,
	}
}

func topologyToProto(data *model.TMutateRequest) *bPb.PutReq {
	extras := map[string]string{}
	extras["user"] = data.Request
	extras["name"] = data.Payload.Name

	tasks := []*bPb.PutTask{}
	for _, region := range data.Payload.Regions {
		for _, cluster := range region.Clusters {
			payload := []*bPb.Payload{}
			for _, node := range cluster.Nodes {
				values := map[string]string{
					"ID":        node.ID,
					"NAME":      node.Name,
					"RETENTION": cluster.Retention,
				}
				for k, v := range node.Labels {
					values[k] = v
				}
				payload = append(payload, &bPb.Payload{
					Value: values,
				})
			}
			tasks = append(tasks, &bPb.PutTask{
				RegionId:  region.ID,
				ClusterId: cluster.ID,
				Selector: &bPb.Selector{
					Labels: data.Payload.Labels,
				},
				Strategy: &bPb.Strategy{
					Type: bPb.StrategyKind_AT_ONCE,
					Kind: "all",
				},
				Payload: payload,
			})
		}
	}

	return &bPb.PutReq{
		Version: data.Version,
		UserId:  data.Request,
		Kind:    tKind(data.Kind),
		Mtdata: &bPb.Metadata{
			TaskName:            data.MTData.TaskName,
			Timestamp:           data.MTData.Timestamp,
			Namespace:           data.MTData.Namespace,
			ForceNamespaceQueue: data.MTData.ForceNSQueue,
			Queue:               data.MTData.Queue,
		},
		Extras: extras,
		Tasks:  tasks,
	}
}

func protoToNodesListResp(data *mPb.ListRsp) *model.NodesResponse {
	rez := &model.NodesResponse{Data: []model.LNode{}}
	for k, val := range data.Data {
		node := model.LNode{
			ID:     k,
			Labels: val.Data,
		}
		rez.Data = append(rez.Data, node)
	}
	return rez
}

func listToProto(data map[string][]string, kind cPb.ReqKind) *cPb.ListReq {
	extras := map[string]string{}
	for k, v := range data {
		if k == labels {
			sort.Strings(v)
			extras[k] = strings.Join(v, ",")
		} else {
			extras[k] = v[0]
		}
	}
	return &cPb.ListReq{
		Extras: extras,
		Kind:   kind,
	}
}

func listNodesToProto(data map[string][]string) *mPb.DataMsg {
	extras := map[string]string{}
	for k, v := range data {
		if k == labels {
			sort.Strings(v)
			extras[k] = strings.Join(v, ",")
		} else {
			extras[k] = v[0]
		}
	}
	return &mPb.DataMsg{
		Data: extras,
	}
}

func protoToNSListResp(resp *cPb.ListResp) *model.NSResponse {
	rez := &model.NSResponse{Result: []model.NSData{}}
	if resp.Data == nil {
		return rez
	}

	for _, lresp := range resp.Data {
		data := model.NSData{
			Age:       lresp.Data["age"],
			Name:      lresp.Data["name"],
			Namespace: lresp.Data["namespace"],
			Labels:    lresp.Data["labels"],
		}
		rez.Result = append(rez.Result, data)
	}
	return rez
}

func protoToSecretsListResp(resp *cPb.ListResp) *model.SecretsResponse {
	rez := &model.SecretsResponse{Result: []model.SecretsData{}}
	if resp.Data == nil {
		return rez
	}

	for _, lresp := range resp.Data {
		data := model.SecretsData{
			RegionId:  lresp.Data["regionid"],
			ClusterId: lresp.Data["clusterid"],
			NodeId:    lresp.Data["nodeid"],
			Secrets:   lresp.Data["secrets"],
		}
		rez.Result = append(rez.Result, data)
	}
	return rez
}

func protoToConfigListResp(resp *cPb.ListResp) *model.ConfigResponse {
	rez := &model.ConfigResponse{Result: []model.ConfigData{}}
	if resp.Data == nil {
		return rez
	}

	for _, lresp := range resp.Data {
		data := model.ConfigData{
			RegionId:  lresp.Data["regionid"],
			ClusterId: lresp.Data["clusterid"],
			NodeId:    lresp.Data["nodeid"],
			Configs:   lresp.Data["configs"],
		}
		rez.Result = append(rez.Result, data)
	}
	return rez
}

func protoToRolesListResp(resp *cPb.ListResp) *model.RolesResponse {
	return &model.RolesResponse{Result: resp.Extras}
}

func protoToActionsListResp(resp *cPb.ListResp) *model.ActionsResponse {
	rez := &model.ActionsResponse{Result: []model.ActionsData{}}
	if resp.Data == nil {
		return rez
	}

	actions := map[string]string{}
	for _, lresp := range resp.Data {
		for k, v := range lresp.Data {
			if strings.HasPrefix(k, "timestamp_") {
				actions[k] = v
			}
		}

		data := model.ActionsData{
			RegionId:  lresp.Data["regionid"],
			ClusterId: lresp.Data["clusterid"],
			NodeId:    lresp.Data["nodeid"],
			Actions:   actions,
		}
		rez.Result = append(rez.Result, data)
	}
	return rez
}

func traceGetToJson(resp *sPb.GetResp) *model.Trace {
	trace := []model.Span{}
	traceId := "no trace"
	for _, item := range resp.Trace {
		traceId = item.SpanContext.TraceId
		trace = append(trace, model.Span{
			Name:      item.Name,
			Logs:      item.Logs,
			Tags:      item.Tags,
			StartTime: item.StartTime,
			EndTime:   item.EndTime,
			Context: model.SpanContext{
				TraceId:       traceId,
				SpanId:        item.SpanContext.SpanId,
				ParrentSpanId: item.SpanContext.ParrentSpanId,
				Baggage:       item.SpanContext.Baggage,
			},
		})
	}

	return &model.Trace{
		TraceId: traceId,
		Trace:   trace,
	}
}

func traceListToJson(resp *sPb.ListResp) *model.Traces {
	traces := []model.Trace{}
	for _, item := range resp.Traces {
		traces = append(traces, *traceGetToJson(item))
	}

	return &model.Traces{
		Traces: traces,
	}
}

func toGetTrace(traceId string) *sPb.GetReq {
	return &sPb.GetReq{
		TraceId: traceId,
	}
}

func toListTrace(tags string) *sPb.ListReq {
	return &sPb.ListReq{
		Query: map[string]string{"query": tags},
	}
}
