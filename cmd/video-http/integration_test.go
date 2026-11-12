package main

import (
 "context"
 "encoding/json"
 "fmt"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"
 "time"
)

func TestMCPHTTPIntegrationWithOAuth(t *testing.T) {
 token:="integration-token"
 introspection:=httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  if r.Method!=http.MethodPost {t.Errorf("method: %s",r.Method)}
  id,secret,ok:=r.BasicAuth()
  if !ok || id!="resource-client" || secret!="test-secret" {t.Error("invalid introspection credentials")}
  if err:=r.ParseForm();err!=nil {t.Error(err)}
  active:=r.Form.Get("token")==token
  w.Header().Set("Content-Type","application/json")
  fmt.Fprintf(w,`{"active":%t,"iss":"https://auth.example","aud":"https://video.example","exp":%d,"scope":"mcp:read"}`,active,time.Now().Add(time.Hour).Unix())
 }))
 defer introspection.Close()
 validator,err:=newIntrospector(introspection.URL,"resource-client","test-secret","https://auth.example","https://video.example","mcp:read")
 if err!=nil {t.Fatal(err)}
 validator.client=introspection.Client()
 server:=httptest.NewServer(handlerWithValidator(strings.Repeat("s",40),"https://video.example","https://auth.example",validator))
 defer server.Close()
 client:=server.Client()
 send:=func(method,path,body,bearer string)(int,http.Header,map[string]any){
  t.Helper()
  req,err:=http.NewRequestWithContext(context.Background(),method,server.URL+path,strings.NewReader(body))
  if err!=nil {t.Fatal(err)}
  if bearer!="" {req.Header.Set("Authorization","Bearer "+bearer)}
  if method=="POST" {
   req.Header.Set("Content-Type","application/json")
   req.Header.Set("Accept","application/json, text/event-stream")
   req.Header.Set("MCP-Protocol-Version",protocolVersion)
  }
  res,err:=client.Do(req)
  if err!=nil {t.Fatal(err)}
  defer res.Body.Close()
  var data map[string]any
  _=json.NewDecoder(res.Body).Decode(&data)
  return res.StatusCode,res.Header,data
 }
 t.Run("discovery",func(t *testing.T){
  status,_,data:=send("GET","/.well-known/oauth-protected-resource","","")
  if status!=200 || data["resource"]!="https://video.example" {t.Fatalf("status=%d data=%v",status,data)}
 })
 t.Run("unauthorized",func(t *testing.T){
  status,headers,_:=send("POST","/mcp",`{"jsonrpc":"2.0","id":1,"method":"ping"}`,"")
  if status!=401 || !strings.Contains(headers.Get("WWW-Authenticate"),"resource_metadata") {t.Fatalf("status=%d challenge=%s",status,headers.Get("WWW-Authenticate"))}
 })
 t.Run("static-token-is-not-fallback",func(t *testing.T){
  status,_,_:=send("POST","/mcp",`{"jsonrpc":"2.0","id":1,"method":"ping"}`,strings.Repeat("s",40))
  if status!=401 {t.Fatalf("status=%d",status)}
 })
 t.Run("initialize",func(t *testing.T){
  status,_,data:=send("POST","/mcp",`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,token)
  if status!=200 {t.Fatalf("status=%d",status)}
  result,ok:=data["result"].(map[string]any)
  if !ok || result["protocolVersion"]!=protocolVersion {t.Fatalf("result=%v",data)}
 })
 t.Run("list-tools",func(t *testing.T){
  status,_,data:=send("POST","/mcp",`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,token)
  if status!=200 {t.Fatalf("status=%d",status)}
  result,ok:=data["result"].(map[string]any)
  if !ok {t.Fatalf("result=%v",data)}
  tools,ok:=result["tools"].([]any)
  if !ok || len(tools)!=6 {t.Fatalf("tools=%v",result)}
  found:=false
  for _,tool:=range tools{
   item,ok:=tool.(map[string]any)
   if ok && item["name"]=="video_models"{found=true}
  }
  if !found {t.Fatal("video_models tool missing")}
 })
 t.Run("call-tool",func(t *testing.T){
  status,_,data:=send("POST","/mcp",`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"video_capabilities","arguments":{}}}`,token)
  if status!=200 || data["result"]==nil {t.Fatalf("status=%d result=%v",status,data)}
 })
 t.Run("notification",func(t *testing.T){
  status,_,data:=send("POST","/mcp",`{"jsonrpc":"2.0","method":"notifications/initialized"}`,token)
  if status!=202 || data!=nil {t.Fatalf("status=%d result=%v",status,data)}
 })
 t.Run("invalid-token",func(t *testing.T){
  status,_,_:=send("POST","/mcp",`{"jsonrpc":"2.0","id":4,"method":"ping"}`,"wrong-token")
  if status!=401 {t.Fatalf("status=%d",status)}
 })
}
