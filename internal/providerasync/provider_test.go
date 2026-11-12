package providerasync

import (
 "context"
 "encoding/json"
 "net/http"
 "net/http/httptest"
 "strings"
 "sync/atomic"
 "testing"
 "time"

 "github.com/Ploos-AS/Ploos-AI-Video-Connector/internal/jobs"
)
func TestAsyncLifecycle(t *testing.T){
 var polls atomic.Int32
 server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  if r.Header.Get("Authorization")!="Bearer secret"{t.Error("missing bearer");w.WriteHeader(401);return}
  w.Header().Set("Content-Type","application/json")
  if r.Method=="POST" && r.URL.Path=="/jobs" {
   var req jobs.Request
   if err:=json.NewDecoder(r.Body).Decode(&req);err!=nil || req.Prompt!="hello"{t.Error("invalid request")}
   w.WriteHeader(202)
   _,_=w.Write([]byte(`{"id":"job-1","status":"queued"}`));return
  }
  if r.Method=="GET" && r.URL.Path=="/jobs/job-1" {
   if polls.Add(1)<2 {_,_=w.Write([]byte(`{"id":"job-1","status":"running"}`))}else{_,_=w.Write([]byte(`{"id":"job-1","status":"completed","url":"https://example.invalid/video.mp4","mime_type":"video/mp4"}`))}
   return
  }
  t.Errorf("unexpected %s %s",r.Method,r.URL.Path)
  w.WriteHeader(404)
 }))
 defer server.Close()
 p,err:=New(server.URL+"/jobs","secret",100*time.Millisecond)
 if err!=nil{t.Fatal(err)}
 ctx,cancel:=context.WithTimeout(context.Background(),time.Second)
 defer cancel()
 result,err:=p.Generate(ctx,jobs.Request{Model:"demo",Prompt:"hello"})
 if err!=nil || result.MIMEType!="video/mp4"{t.Fatalf("%+v %v",result,err)}
}
func TestConfiguration(t *testing.T){
 for _,endpoint:=range []string{"https://127.0.0.1:9000/jobs","http://8.8.8.8:9000/jobs","http://example.com:9000/jobs","http://127.0.0.1:9000/other","http://127.0.0.1:9000/jobs?x=1"}{
  if _,err:=New(endpoint,"secret",time.Second);err==nil{t.Errorf("accepted %s",endpoint)}
 }
 if _,err:=New("http://127.0.0.1:9000/jobs","",time.Second);err==nil{t.Fatal("accepted missing token")}
 if _,err:=New("http://127.0.0.1:9000/jobs","secret",time.Millisecond);err==nil{t.Fatal("accepted excessive polling")}
}
func TestCancelRemote(t *testing.T){
 cancelled:=make(chan struct{},1)
 server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  if r.Method=="DELETE" {cancelled<-struct{}{};_,_=w.Write([]byte(`{"id":"job-2","status":"cancelled"}`));return}
  if r.Method=="POST"{_,_=w.Write([]byte(`{"id":"job-2","status":"queued"}`));return}
  _,_=w.Write([]byte(`{"id":"job-2","status":"running"}`))
 }))
 defer server.Close()
 p,err:=New(server.URL+"/jobs","secret",100*time.Millisecond)
 if err!=nil{t.Fatal(err)}
 ctx,cancel:=context.WithCancel(context.Background())
 go func(){time.Sleep(30*time.Millisecond);cancel()}()
 _,err=p.Generate(ctx,jobs.Request{Model:"demo",Prompt:strings.Repeat("x",2)})
 if err==nil{t.Fatal("expected cancellation")}
 select{case <-cancelled:case <-time.After(time.Second):t.Fatal("remote cancel not sent")}
}
