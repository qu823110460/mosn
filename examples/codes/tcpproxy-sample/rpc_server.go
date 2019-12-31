package main

import (
	"fmt"
	"net"
	"time"

	"mosn.io/mosn/pkg/buffer"
	"mosn.io/mosn/pkg/protocol/rpc/sofarpc"
	"mosn.io/mosn/pkg/protocol/rpc/sofarpc/codec"
)

type SofaRPCServer struct {
	Listener net.Listener
}

func (s *SofaRPCServer) Run() {
	for {
		conn, err := s.Listener.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				fmt.Printf("[RPC Server] Accept temporary error: %v\n", ne)
				continue
			}
			return //not temporary error, exit
		}
		fmt.Println("[RPC Server] get connection :", conn.RemoteAddr().String())
		go s.Serve(conn)
	}
}

func (s *SofaRPCServer) Serve(conn net.Conn) {
	iobuf := buffer.NewIoBuffer(102400)
	for {
		now := time.Now()
		conn.SetReadDeadline(now.Add(30 * time.Second))
		buf := make([]byte, 10*1024)
		bytesRead, err := conn.Read(buf)
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Timeout() {
				fmt.Printf("[RPC Server] Connect read error: %v\n", err)
				continue
			}

		}
		if bytesRead > 0 {
			iobuf.Write(buf[:bytesRead])
			for iobuf.Len() > 1 {
				cmd, _ := codec.BoltCodec.Decode(nil, iobuf)
				if cmd == nil {
					break
				}
				if req, ok := cmd.(*sofarpc.BoltRequest); ok {
					resp := buildBoltV1Response(req)
					iobufresp, err := codec.BoltCodec.Encode(nil, resp)
					if err != nil {
						fmt.Printf("[RPC Server] build response error: %v\n", err)
					} else {
						fmt.Printf("[RPC Server] reponse connection: %s, requestId: %d\n", conn.RemoteAddr().String(), resp.RequestID())
						respdata := iobufresp.Bytes()
						conn.Write(respdata)
					}
				}
			}
		}
	}
}

func buildBoltV1Response(req *sofarpc.BoltRequest) *sofarpc.BoltResponse {
	return &sofarpc.BoltResponse{
		Protocol:       req.Protocol,
		CmdType:        sofarpc.RESPONSE,
		CmdCode:        sofarpc.RPC_RESPONSE,
		Version:        req.Version,
		ReqID:          req.ReqID,
		Codec:          req.Codec,
		ResponseStatus: sofarpc.RESPONSE_STATUS_SUCCESS,
		HeaderLen:      req.HeaderLen,
		HeaderMap:      req.HeaderMap,
	}

}

func main() {
	ln, err := net.Listen("tcp", "127.0.0.1:8080")
	if err != nil {
		fmt.Println(err)
		return
	}
	server := &SofaRPCServer{ln}
	server.Run()
}
