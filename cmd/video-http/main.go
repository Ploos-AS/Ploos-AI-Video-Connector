// video-http exposes the diagnostic MCP tool over a restricted HTTP endpoint.
// This is an experimental stateless JSON-response subset of Streamable HTTP.
package main

import (
 "crypto/subtle"
 "encoding/json"
 "flag"
 "fmt"
 "log"
 "net/http"
 "os"
 "strings"
 "time"
)

type rpc struct { JSONRPC string `json:"jsonrpc"`; ID json.RawMessage `json:"id,omitempty"`; Method string `json:"method"`; Params json.RawMessage `json:"params,omitempty"` }
func result(id json.RawMessage,v any) any{return map[string]any{"jsonrpc":"2.0","id":id,"result":v}}
func rpcError(id json.RawMessage,code int,message string) any{return map[string]any{"jsonrpc":"2.0","id":id,"error":map[string]any{"code":code,"message":message}}}
func handler(token string) http.Handler {
 return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  if r.URL.Path!="/mcp" {http.NotFound(w,r);return}
  if r.Method!=http.MethodPost {w.Header().Set("Allow","POST");http.Error(w,"method not allowed",405);return}
  // Block cross-site browser requests. Non-browser clients may omit Origin.
  if r.Header.Get("Origin")!="" {http.Error(w,"browser origins not allowed",403);return}
  authorization:=r.Header.Get("Authorization")
  expected:="Bearer "+token
  if subtle.ConstantTimeCompare([]byte(authorization),[]byte(expected))!=1 {w.Header().Set("WWW-Authenticate","Bearer");http.Error(w,"unauthorized",401);return}
  r.Body=http.MaxBytesReader(w,r.Body,1<<20)
  defer r.Body.Close()
  var q rpc
  if err:=json.NewDecoder(r.Body).Decode(&q);err!=nil {http.Error(w,"invalid JSON",400);return}
  if q.JSONRPC!="2.0" {http.Error(w,"invalid JSON-RPC version",400);return}
  if len(q.ID)==0 {w.WriteHeader(http.StatusAccepted);return}
  var response any
  switch q.Method {
  case "initialize":
   response=result(q.ID,map[string]any{"protocolVersion":"2024-11-05","capabilities":map[string]any{"tools":map[string]any{}},"serverInfo":map[string]string{"name":"ploos-ai-video-connector","version":"0.1.0-dev"}})
  case "ping":response=result(q.ID,map[string]any{})
  case "tools/list":response=result(q.ID,map[string]any{"tools":[]any{map[string]any{"name":"video_capabilities","description":"Report connector capabilities without paid calls","inputSchema":map[string]any{"type":"object","properties":map[string]any{}}}}})
  case "tools/call":
   var p struct{Name string `json:"name"`}
   _=json.Unmarshal(q.Params,&p)
   if p.Name=="video_capabilities" {response=result(q.ID,map[string]any{"content":[]any{map[string]string{"type":"text","text":"Diagnostic only: no video providers configured."}}})}else{response=rpcError(q.ID,-32602,"unknown tool")}
  default:response=rpcError(q.ID,-32601,"method not found")
  }
  w.Header().Set("Content-Type","application/json")
  w.Header().Set("Cache-Control","no-store")
  _=json.NewEncoder(w).Encode(response)
 })
}
func main(){
 address:=flag.String("listen","127.0.0.1:8787","HTTP listen address (use a TLS reverse proxy for remote access)")
 flag.Parse()
 token:=strings.TrimSpace(os.Getenv("PLOOS_MCP_BEARER_TOKEN"))
 if len(token)<32 {log.Fatal("PLOOS_MCP_BEARER_TOKEN must contain at least 32 characters")}
 srv:=&http.Server{Addr:*address,Handler:handler(token),ReadHeaderTimeout:5*time.Second,ReadTimeout:15*time.Second,WriteTimeout:15*time.Second}
 fmt.Fprintf(os.Stderr,"diagnostic HTTP MCP listening on %s\n",*address)
 log.Fatal(srv.ListenAndServe())
}
