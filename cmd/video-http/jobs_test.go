package main

import (
 "bytes"
 "encoding/json"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"
)
func TestVideoJobsFailClosedWithoutProvider(t *testing.T) {
 token:=strings.Repeat("t",40)
 server:=httptest.NewServer(handler(token))
 defer server.Close()
 for _,tc:=range []struct{tool,arguments string;code float64}{
  {"video_create",`{"model":"demo","prompt":"test"}`,-32001},
  {"video_status",`{"id":"missing"}`,-32602},
  {"video_cancel",`{"id":"missing"}`,-32602},
 } {
  payload:=map[string]any{"jsonrpc":"2.0","id":1,"method":"tools/call","params":map[string]any{"name":tc.tool,"arguments":json.RawMessage(tc.arguments)}}
  body,_:=json.Marshal(payload)
  req,err:=http.NewRequest("POST",server.URL+"/mcp",bytes.NewReader(body))
  if err!=nil {t.Fatal(err)}
  req.Header.Set("Authorization","Bearer "+token)
  req.Header.Set("Content-Type","application/json")
  req.Header.Set("Accept","application/json, text/event-stream")
  res,err:=server.Client().Do(req)
  if err!=nil {t.Fatal(err)}
  var data struct{Error struct{Code float64 `json:"code"`} `json:"error"`}
  err=json.NewDecoder(res.Body).Decode(&data)
  res.Body.Close()
  if err!=nil || data.Error.Code!=tc.code {t.Fatalf("%s: %+v, %v",tc.tool,data,err)}
 }
}
