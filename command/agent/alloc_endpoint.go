package agent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang/snappy"
	cstructs "github.com/hashicorp/nomad/client/structs"
	"github.com/hashicorp/nomad/nomad/structs"
)

const (
	allocNotFoundErr    = "allocation not found"
	resourceNotFoundErr = "resource not found"
)

func (s *HTTPServer) AllocsRequest(resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	if req.Method != "GET" {
		return nil, CodedError(405, ErrInvalidMethod)
	}

	args := structs.AllocListRequest{}
	if s.parse(resp, req, &args.Region, &args.QueryOptions) {
		return nil, nil
	}

	var out structs.AllocListResponse
	if err := s.agent.RPC("Alloc.List", &args, &out); err != nil {
		return nil, err
	}

	setMeta(resp, &out.QueryMeta)
	if out.Allocations == nil {
		out.Allocations = make([]*structs.AllocListStub, 0)
	}
	for _, alloc := range out.Allocations {
		alloc.SetEventDisplayMessages()
	}
	return out.Allocations, nil
}

func (s *HTTPServer) AllocSpecificRequest(resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	reqSuffix := strings.TrimPrefix(req.URL.Path, "/v1/allocation/")

	// tokenize the suffix of the path to get the alloc id and find the action
	// invoked on the alloc id
	tokens := strings.Split(reqSuffix, "/")
	if len(tokens) > 2 || len(tokens) < 1 {
		return nil, CodedError(404, resourceNotFoundErr)
	}
	allocID := tokens[0]

	if len(tokens) == 1 {
		return s.allocGet(allocID, resp, req)
	}

	switch tokens[1] {
	case "stop":
		return s.allocStop(allocID, resp, req)
	}

	return nil, CodedError(404, resourceNotFoundErr)
}

func (s *HTTPServer) allocGet(allocID string, resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	if req.Method != "GET" {
		return nil, CodedError(405, ErrInvalidMethod)
	}

	args := structs.AllocSpecificRequest{
		AllocID: allocID,
	}
	if s.parse(resp, req, &args.Region, &args.QueryOptions) {
		return nil, nil
	}

	var out structs.SingleAllocResponse
	if err := s.agent.RPC("Alloc.GetAlloc", &args, &out); err != nil {
		return nil, err
	}

	setMeta(resp, &out.QueryMeta)
	if out.Alloc == nil {
		return nil, CodedError(404, "alloc not found")
	}

	// Decode the payload if there is any
	alloc := out.Alloc
	if alloc.Job != nil && len(alloc.Job.Payload) != 0 {
		decoded, err := snappy.Decode(nil, alloc.Job.Payload)
		if err != nil {
			return nil, err
		}
		alloc = alloc.Copy()
		alloc.Job.Payload = decoded
	}
	alloc.SetEventDisplayMessages()

	return alloc, nil
}

func (s *HTTPServer) allocStop(allocID string, resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	if !(req.Method == "POST" || req.Method == "PUT") {
		return nil, CodedError(405, ErrInvalidMethod)
	}

	sr := &structs.AllocStopRequest{
		AllocID: allocID,
	}
	s.parseWriteRequest(req, &sr.WriteRequest)

	var out structs.AllocStopResponse
	err := s.agent.RPC("Alloc.Stop", &sr, &out)
	return &out, err
}

func (s *HTTPServer) ClientAllocRequest(resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	reqSuffix := strings.TrimPrefix(req.URL.Path, "/v1/client/allocation/")

	// tokenize the suffix of the path to get the alloc id and find the action
	// invoked on the alloc id
	tokens := strings.Split(reqSuffix, "/")
	if len(tokens) != 2 {
		return nil, CodedError(404, resourceNotFoundErr)
	}
	allocID := tokens[0]
	switch tokens[1] {
	case "stats":
		return s.allocStats(allocID, resp, req)
	case "snapshot":
		if s.agent.client == nil {
			return nil, clientNotRunning
		}
		return s.allocSnapshot(allocID, resp, req)
	case "restart":
		return s.allocRestart(allocID, resp, req)
	case "gc":
		return s.allocGC(allocID, resp, req)
	case "signal":
		return s.allocSignal(allocID, resp, req)
	}

	return nil, CodedError(404, resourceNotFoundErr)
}

func (s *HTTPServer) ClientGCRequest(resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	// Get the requested Node ID
	requestedNode := req.URL.Query().Get("node_id")

	// Build the request and parse the ACL token
	args := structs.NodeSpecificRequest{
		NodeID: requestedNode,
	}
	s.parse(resp, req, &args.QueryOptions.Region, &args.QueryOptions)

	// Determine the handler to use
	useLocalClient, useClientRPC, useServerRPC := s.rpcHandlerForNode(requestedNode)

	// Make the RPC
	var reply structs.GenericResponse
	var rpcErr error
	if useLocalClient {
		rpcErr = s.agent.Client().ClientRPC("Allocations.GarbageCollectAll", &args, &reply)
	} else if useClientRPC {
		rpcErr = s.agent.Client().RPC("ClientAllocations.GarbageCollectAll", &args, &reply)
	} else if useServerRPC {
		rpcErr = s.agent.Server().RPC("ClientAllocations.GarbageCollectAll", &args, &reply)
	} else {
		rpcErr = CodedError(400, "No local Node and node_id not provided")
	}

	if rpcErr != nil {
		if structs.IsErrNoNodeConn(rpcErr) {
			rpcErr = CodedError(404, rpcErr.Error())
		}
	}

	return nil, rpcErr
}

func (s *HTTPServer) allocRestart(allocID string, resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	// Build the request and parse the ACL token
	args := structs.AllocRestartRequest{
		AllocID:  allocID,
		TaskName: "",
	}
	s.parse(resp, req, &args.QueryOptions.Region, &args.QueryOptions)

	// Explicitly parse the body separately to disallow overriding AllocID in req Body.
	var reqBody struct {
		TaskName string
	}
	err := json.NewDecoder(req.Body).Decode(&reqBody)
	if err != nil {
		return nil, err
	}
	if reqBody.TaskName != "" {
		args.TaskName = reqBody.TaskName
	}

	// Determine the handler to use
	useLocalClient, useClientRPC, useServerRPC := s.rpcHandlerForAlloc(allocID)

	// Make the RPC
	var reply structs.GenericResponse
	var rpcErr error
	if useLocalClient {
		rpcErr = s.agent.Client().ClientRPC("Allocations.Restart", &args, &reply)
	} else if useClientRPC {
		rpcErr = s.agent.Client().RPC("ClientAllocations.Restart", &args, &reply)
	} else if useServerRPC {
		rpcErr = s.agent.Server().RPC("ClientAllocations.Restart", &args, &reply)
	} else {
		rpcErr = CodedError(400, "No local Node and node_id not provided")
	}

	if rpcErr != nil {
		if structs.IsErrNoNodeConn(rpcErr) || structs.IsErrUnknownAllocation(rpcErr) {
			rpcErr = CodedError(404, rpcErr.Error())
		}
	}

	return reply, rpcErr
}

func (s *HTTPServer) allocGC(allocID string, resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	// Build the request and parse the ACL token
	args := structs.AllocSpecificRequest{
		AllocID: allocID,
	}
	s.parse(resp, req, &args.QueryOptions.Region, &args.QueryOptions)

	// Determine the handler to use
	useLocalClient, useClientRPC, useServerRPC := s.rpcHandlerForAlloc(allocID)

	// Make the RPC
	var reply structs.GenericResponse
	var rpcErr error
	if useLocalClient {
		rpcErr = s.agent.Client().ClientRPC("Allocations.GarbageCollect", &args, &reply)
	} else if useClientRPC {
		rpcErr = s.agent.Client().RPC("ClientAllocations.GarbageCollect", &args, &reply)
	} else if useServerRPC {
		rpcErr = s.agent.Server().RPC("ClientAllocations.GarbageCollect", &args, &reply)
	} else {
		rpcErr = CodedError(400, "No local Node and node_id not provided")
	}

	if rpcErr != nil {
		if structs.IsErrNoNodeConn(rpcErr) || structs.IsErrUnknownAllocation(rpcErr) {
			rpcErr = CodedError(404, rpcErr.Error())
		}
	}

	return nil, rpcErr
}

func (s *HTTPServer) allocSignal(allocID string, resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	if !(req.Method == "POST" || req.Method == "PUT") {
		return nil, CodedError(405, ErrInvalidMethod)
	}

	// Build the request and parse the ACL token
	args := structs.AllocSignalRequest{}
	err := decodeBody(req, &args)
	if err != nil {
		return nil, CodedError(400, fmt.Sprintf("Failed to decode body: %v", err))
	}
	s.parse(resp, req, &args.QueryOptions.Region, &args.QueryOptions)
	args.AllocID = allocID

	// Determine the handler to use
	useLocalClient, useClientRPC, useServerRPC := s.rpcHandlerForAlloc(allocID)

	// Make the RPC
	var reply structs.GenericResponse
	var rpcErr error
	if useLocalClient {
		rpcErr = s.agent.Client().ClientRPC("Allocations.Signal", &args, &reply)
	} else if useClientRPC {
		rpcErr = s.agent.Client().RPC("ClientAllocations.Signal", &args, &reply)
	} else if useServerRPC {
		rpcErr = s.agent.Server().RPC("ClientAllocations.Signal", &args, &reply)
	} else {
		rpcErr = CodedError(400, "No local Node and node_id not provided")
	}

	if rpcErr != nil {
		if structs.IsErrNoNodeConn(rpcErr) || structs.IsErrUnknownAllocation(rpcErr) {
			rpcErr = CodedError(404, rpcErr.Error())
		}
	}

	return reply, rpcErr
}

func (s *HTTPServer) allocSnapshot(allocID string, resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	var secret string
	s.parseToken(req, &secret)
	if !s.agent.Client().ValidateMigrateToken(allocID, secret) {
		return nil, structs.ErrPermissionDenied
	}

	allocFS, err := s.agent.Client().GetAllocFS(allocID)
	if err != nil {
		return nil, fmt.Errorf(allocNotFoundErr)
	}
	if err := allocFS.Snapshot(resp); err != nil {
		return nil, fmt.Errorf("error making snapshot: %v", err)
	}
	return nil, nil
}

func (s *HTTPServer) allocStats(allocID string, resp http.ResponseWriter, req *http.Request) (interface{}, error) {

	// Build the request and parse the ACL token
	task := req.URL.Query().Get("task")
	args := cstructs.AllocStatsRequest{
		AllocID: allocID,
		Task:    task,
	}
	s.parse(resp, req, &args.QueryOptions.Region, &args.QueryOptions)

	// Determine the handler to use
	useLocalClient, useClientRPC, useServerRPC := s.rpcHandlerForAlloc(allocID)

	// Make the RPC
	var reply cstructs.AllocStatsResponse
	var rpcErr error
	if useLocalClient {
		rpcErr = s.agent.Client().ClientRPC("Allocations.Stats", &args, &reply)
	} else if useClientRPC {
		rpcErr = s.agent.Client().RPC("ClientAllocations.Stats", &args, &reply)
	} else if useServerRPC {
		rpcErr = s.agent.Server().RPC("ClientAllocations.Stats", &args, &reply)
	} else {
		rpcErr = CodedError(400, "No local Node and node_id not provided")
	}

	if rpcErr != nil {
		if structs.IsErrNoNodeConn(rpcErr) || structs.IsErrUnknownAllocation(rpcErr) {
			rpcErr = CodedError(404, rpcErr.Error())
		}
	}

	return reply.Stats, rpcErr
}
