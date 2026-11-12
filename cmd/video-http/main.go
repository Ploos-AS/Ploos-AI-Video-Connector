// video-http implements a stateless MCP Streamable HTTP JSON-response endpoint.
// Remote deployment requires TLS termination and an authorization design appropriate
// for the client. This diagnostic server uses a static bearer token only.
package main

import (
 "crypto/subtle"
 "errors"
 "github.com/Ploos-AS/Ploos-AI-Video-Connector/internal/jobs"
 "github.com/Ploos-AS/Ploos-AI-Video-Connector/internal/providerhttp"
 "github.com/Ploos-AS/Ploos-AI-Video-Connector/internal/providerasync"
 "encoding/json"
 "flag"
 "fmt"
 "log"
 "net/http"
 "net/url"
 "os"
 "strings"
 "time"
)

const protocolVersion = "2025-03-26"

type rpc struct {
 JSONRPC string `json:"jsonrpc"`
 ID json.RawMessage `json:"id,omitempty"`
 Method string `json:"method"`
 Params json.RawMessage `json:"params,omitempty"`
}
func result(id json.RawMessage,v any) any{return map[string]any{"jsonrpc":"2.0","id":id,"result":v}}
func rpcError(id json.RawMessage,code int,message string) any{return map[string]any{"jsonrpc":"2.0","id":id,"error":map[string]any{"code":code,"message":message}}}
func accepts(h,media string) bool {
 for _,part:=range strings.Split(h,",") {
  t:=strings.TrimSpace(strings.SplitN(part,";",2)[0])
  if t==media || t=="*/*" {return true}
 }
 return false
}
func handler(token string) http.Handler {
 return handlerWithOAuthMetadata(token, "", "")
}

// Discovery only: OAuth token validation is not implemented.
func handlerWithOAuthMetadata(token, resource, issuer string) http.Handler {
 return handlerWithValidator(token, resource, issuer, nil)
}
func handlerWithValidator(token, resource, issuer string, validator *introspector) http.Handler {
 manager,_:=jobs.New(nil,2,100,5*time.Minute)
 return handlerWithJobs(token,resource,issuer,validator,manager)
}
func handlerWithJobs(token, resource, issuer string, validator *introspector, manager *jobs.Manager) http.Handler {
 metadataEnabled := validHTTPS(resource) && validHTTPS(issuer)
 return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  if r.URL.Path=="/.well-known/oauth-protected-resource" {
   if !metadataEnabled {http.NotFound(w,r);return}
   if r.Method!=http.MethodGet {w.Header().Set("Allow","GET");http.Error(w,"method not allowed",405);return}
   w.Header().Set("Content-Type","application/json")
   w.Header().Set("Cache-Control","no-store")
   _=json.NewEncoder(w).Encode(map[string]any{
    "resource":resource,
    "authorization_servers":[]string{issuer},
    "bearer_methods_supported":[]string{"header"},
   })
   return
  }
  if r.URL.Path!="/mcp" {http.NotFound(w,r);return}
  w.Header().Set("Cache-Control","no-store")
  // Do not allow cross-origin browser requests; non-browser MCP clients omit Origin.
  if r.Header.Get("Origin")!="" {http.Error(w,"browser origins not allowed",http.StatusForbidden);return}
  authorization:=r.Header.Get("Authorization")
  authorized:=false
  if validator!=nil {
   if strings.HasPrefix(authorization,"Bearer ") {
    authorized=validator.validate(r.Context(),strings.TrimPrefix(authorization,"Bearer "))==nil
   }
  } else {
   expected:="Bearer "+token
   authorized=subtle.ConstantTimeCompare([]byte(authorization),[]byte(expected))==1
  }
  if !authorized {
   challenge := "Bearer"
   if metadataEnabled {challenge = fmt.Sprintf("Bearer resource_metadata=%q", strings.TrimRight(resource,"/")+"/.well-known/oauth-protected-resource")}
   w.Header().Set("WWW-Authenticate",challenge)
   http.Error(w,"unauthorized",http.StatusUnauthorized);return
  }
  if r.Method!=http.MethodPost {
   w.Header().Set("Allow","POST")
   http.Error(w,"method not allowed (stateless server)",http.StatusMethodNotAllowed);return
  }
  if !accepts(r.Header.Get("Content-Type"),"application/json") {
   http.Error(w,"Content-Type must be application/json",http.StatusUnsupportedMediaType);return
  }
  accept:=r.Header.Get("Accept")
  if !accepts(accept,"application/json") || !accepts(accept,"text/event-stream") {
   http.Error(w,"Accept must include application/json and text/event-stream",http.StatusNotAcceptable);return
  }
  if version:=r.Header.Get("MCP-Protocol-Version");version!="" && version!=protocolVersion {
   http.Error(w,"unsupported MCP protocol version",http.StatusBadRequest);return
  }
  r.Body=http.MaxBytesReader(w,r.Body,1<<20)
  defer r.Body.Close()
  var q rpc
  decoder:=json.NewDecoder(r.Body)
  if err:=decoder.Decode(&q);err!=nil {http.Error(w,"invalid JSON",http.StatusBadRequest);return}
  if q.JSONRPC!="2.0" || q.Method=="" {http.Error(w,"invalid JSON-RPC request",http.StatusBadRequest);return}
  // MCP notifications and responses contain no ID and receive 202 without a body.
  if len(q.ID)==0 {w.WriteHeader(http.StatusAccepted);return}
  var response any
  switch q.Method {
  case "initialize":
   response=result(q.ID,map[string]any{
    "protocolVersion":protocolVersion,
    "capabilities":map[string]any{"tools":map[string]any{}},
    "serverInfo":map[string]string{"name":"ploos-ai-video-connector","version":"0.1.0-dev"},
   })
  case "ping":response=result(q.ID,map[string]any{})
  case "tools/list":
   response=result(q.ID,map[string]any{"tools":[]any{map[string]any{
    "name":"video_capabilities",
    "description":"Report connector capabilities without paid calls",
    "inputSchema":map[string]any{"type":"object","properties":map[string]any{},"additionalProperties":false},
   },map[string]any{"name":"video_create","description":"Create a video job (requires configured provider)","inputSchema":map[string]any{"type":"object","properties":map[string]any{"model":map[string]any{"type":"string"},"prompt":map[string]any{"type":"string"}},"required":[]string{"model","prompt"}}},map[string]any{"name":"video_status","description":"Get video job status and result","inputSchema":map[string]any{"type":"object","properties":map[string]any{"id":map[string]any{"type":"string"}},"required":[]string{"id"}}},map[string]any{"name":"video_cancel","description":"Cancel video job","inputSchema":map[string]any{"type":"object","properties":map[string]any{"id":map[string]any{"type":"string"}},"required":[]string{"id"}}}}})
  case "tools/call":
   var p struct{
    Name string `json:"name"`
    Arguments json.RawMessage `json:"arguments"`
   }
   _=json.Unmarshal(q.Params,&p)
   switch p.Name {
   case "video_capabilities":
    response=result(q.ID,map[string]any{"content":[]any{map[string]string{"type":"text","text":"Video job API available; no provider configured."}}})
   case "video_create":
    var args jobs.Request
    if json.Unmarshal(p.Arguments,&args)!=nil {response=rpcError(q.ID,-32602,"invalid arguments");break}
    job,err:=manager.Submit(args)
    if err!=nil {
     if errors.Is(err,jobs.ErrStorage){response=rpcError(q.ID,-32603,"video job storage unavailable")
     }else if errors.Is(err,jobs.ErrUnavailable){response=rpcError(q.ID,-32001,"video provider unavailable")
     }else if errors.Is(err,jobs.ErrBusy){response=rpcError(q.ID,-32002,"video job capacity reached")
     }else{response=rpcError(q.ID,-32602,"invalid video request")}
    }else{response=result(q.ID,map[string]any{"structuredContent":job})}
   case "video_status","video_cancel":
    var args struct{ID string `json:"id"`}
    if json.Unmarshal(p.Arguments,&args)!=nil || args.ID=="" {response=rpcError(q.ID,-32602,"invalid job id");break}
    var job jobs.Job
    var err error
    if p.Name=="video_cancel"{job,err=manager.Cancel(args.ID)}else{job,err=manager.Get(args.ID)}
    if errors.Is(err,jobs.ErrStorage){response=rpcError(q.ID,-32603,"video job storage unavailable")}else if err!=nil {response=rpcError(q.ID,-32602,"job not found")}else{response=result(q.ID,map[string]any{"structuredContent":job})}
   default:response=rpcError(q.ID,-32602,"unknown tool")
   }
  default:response=rpcError(q.ID,-32601,"method not found")
  }
  w.Header().Set("Content-Type","application/json")
  _=json.NewEncoder(w).Encode(response)
 })
}
func validHTTPS(raw string) bool {
 u,err:=url.Parse(raw)
 return err==nil && u.Scheme=="https" && u.Host!="" && u.User==nil && u.RawQuery=="" && u.Fragment=="" && u.Path==""
}
func main(){
 address:=flag.String("listen","127.0.0.1:8787","HTTP listen address (TLS reverse proxy required for remote access)")
 flag.Parse()
 token:=strings.TrimSpace(os.Getenv("PLOOS_MCP_BEARER_TOKEN"))
 var validator *introspector
 if endpoint:=strings.TrimSpace(os.Getenv("PLOOS_OAUTH_INTROSPECTION_URL"));endpoint!="" {
  var err error
  validator,err=newIntrospector(endpoint,os.Getenv("PLOOS_OAUTH_CLIENT_ID"),os.Getenv("PLOOS_OAUTH_CLIENT_SECRET"),os.Getenv("PLOOS_OAUTH_ISSUER"),os.Getenv("PLOOS_MCP_PUBLIC_URL"),os.Getenv("PLOOS_OAUTH_REQUIRED_SCOPE"))
  if err!=nil {log.Fatal(err)}
 } else if len(token)<32 {log.Fatal("PLOOS_MCP_BEARER_TOKEN must contain at least 32 characters")}
 var provider jobs.Provider
 asyncEndpoint:=strings.TrimSpace(os.Getenv("PLOOS_VIDEO_ASYNC_ENDPOINT"))
 httpEndpoint:=strings.TrimSpace(os.Getenv("PLOOS_VIDEO_HTTP_ENDPOINT"))
 if asyncEndpoint!="" && httpEndpoint!="" {log.Fatal("configure only one video provider")}
 if asyncEndpoint!="" {
  var err error
  provider,err=providerasync.New(asyncEndpoint,os.Getenv("PLOOS_VIDEO_ASYNC_TOKEN"),2*time.Second)
  if err!=nil {log.Fatal(err)}
 }
 if endpoint:=httpEndpoint;endpoint!="" {
  var err error
  provider,err=providerhttp.New(endpoint,os.Getenv("PLOOS_VIDEO_HTTP_TOKEN"))
  if err!=nil {log.Fatal(err)}
 }
 manager,err:=jobs.New(provider,2,100,2*time.Minute)
 if err!=nil {log.Fatal(err)}
 if dbPath:=strings.TrimSpace(os.Getenv("PLOOS_VIDEO_SQLITE_PATH"));dbPath!="" {
  store,err:=jobs.OpenSQLite(dbPath)
  if err!=nil {log.Fatal(err)}
  defer store.Close()
  if err:=manager.AttachStore(store);err!=nil {log.Fatal(err)}
 }
 go func(){
  ticker:=time.NewTicker(time.Hour)
  defer ticker.Stop()
  for range ticker.C {
   if _,err:=manager.Prune(time.Now().UTC().Add(-7*24*time.Hour));err!=nil {log.Printf("video job pruning failed: %v",err)}
  }
 }()
 srv:=&http.Server{Addr:*address,Handler:handlerWithJobs(token,strings.TrimSpace(os.Getenv("PLOOS_MCP_PUBLIC_URL")),strings.TrimSpace(os.Getenv("PLOOS_OAUTH_ISSUER")),validator,manager),ReadHeaderTimeout:5*time.Second,ReadTimeout:15*time.Second,WriteTimeout:15*time.Second}
 fmt.Fprintf(os.Stderr,"stateless diagnostic MCP HTTP listening on %s\n",*address)
 log.Fatal(srv.ListenAndServe())
}
