package main

import (
 "bufio"
 "encoding/json"
 "flag"
 "fmt"
 "os"
)

type request struct { JSONRPC string `json:"jsonrpc"`; ID json.RawMessage `json:"id,omitempty"`; Method string `json:"method"`; Params json.RawMessage `json:"params,omitempty"` }
func respond(id json.RawMessage, result any) {
 out,_:=json.Marshal(map[string]any{"jsonrpc":"2.0","id":id,"result":result})
 fmt.Println(string(out))
}
func main(){
 transport:=flag.String("transport","stdio","MCP transport (stdio)")
 flag.Parse()
 if *transport!="stdio" {fmt.Fprintln(os.Stderr,"only stdio is implemented");os.Exit(2)}
 scanner:=bufio.NewScanner(os.Stdin)
 scanner.Buffer(make([]byte,4096), 4*1024*1024)
 for scanner.Scan(){
  var req request
  if json.Unmarshal(scanner.Bytes(),&req)!=nil || req.Method=="" {continue}
  if len(req.ID)==0 {continue}
  switch req.Method {
  case "initialize":
   respond(req.ID,map[string]any{"protocolVersion":"2024-11-05","capabilities":map[string]any{"tools":map[string]any{}},"serverInfo":map[string]string{"name":"ploos-ai-video-connector","version":"0.1.0-dev"}})
  case "ping":respond(req.ID,map[string]any{})
  case "tools/list":
   respond(req.ID,map[string]any{"tools":[]any{map[string]any{"name":"video_capabilities","description":"Report current connector capabilities without making paid API calls","inputSchema":map[string]any{"type":"object","properties":map[string]any{},"additionalProperties":false}}}})
  case "tools/call":
   var p struct{Name string `json:"name"`}
   _=json.Unmarshal(req.Params,&p)
   if p.Name=="video_capabilities" {respond(req.ID,map[string]any{"content":[]any{map[string]string{"type":"text","text":"MCP stdio available; video generation and providers not yet implemented."}}})} else {sendError(req.ID,-32602,"Unknown tool")}
  default:sendError(req.ID,-32601,"Method not found")
  }
 }
 if err:=scanner.Err();err!=nil {fmt.Fprintln(os.Stderr,err);os.Exit(1)}
}
func sendError(id json.RawMessage,code int,msg string){out,_:=json.Marshal(map[string]any{"jsonrpc":"2.0","id":id,"error":map[string]any{"code":code,"message":msg}});fmt.Println(string(out))}
