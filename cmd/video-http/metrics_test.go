package main

import (
 "net/http/httptest"
 "strings"
 "testing"
 "time"

 "github.com/Ploos-AS/Ploos-AI-Video-Connector/internal/jobs"
)
func TestMetricsRequiresLoopbackAndBearer(t *testing.T){
 m,err:=jobs.New(nil,1,10,time.Minute)
 if err!=nil{t.Fatal(err)}
 h:=metricsHandler(strings.Repeat("a",32),m)
 test:=func(remote,auth string,want int)string{
  t.Helper()
  r:=httptest.NewRequest("GET","/metrics",nil)
  r.RemoteAddr=remote
  r.Header.Set("Authorization",auth)
  w:=httptest.NewRecorder()
  h.ServeHTTP(w,r)
  if w.Code!=want{t.Fatalf("remote=%q status=%d want=%d",remote,w.Code,want)}
  return w.Body.String()
 }
 test("198.51.100.2:1234","Bearer "+strings.Repeat("a",32),403)
 test("127.0.0.1:1234","",401)
 body:=test("127.0.0.1:1234","Bearer "+strings.Repeat("a",32),200)
 if !strings.Contains(body,"ploos_video_jobs{status=\"completed\"} 0"){t.Fatal(body)}
 if strings.Contains(body,"prompt") || strings.Contains(body,"remote_id"){t.Fatal("sensitive fields in metrics")}
}
