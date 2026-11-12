package main

import (
 "crypto/subtle"
 "fmt"
 "net"
 "net/http"
 "strconv"
 "strings"
 "time"

 "github.com/Ploos-AS/Ploos-AI-Video-Connector/internal/jobs"
)

func metricsHandler(token string,manager *jobs.Manager)http.Handler{
 return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  if r.URL.Path!="/metrics"{http.NotFound(w,r);return}
  if r.Method!=http.MethodGet{w.Header().Set("Allow","GET");http.Error(w,"method not allowed",405);return}
  // Defense in depth: only loopback clients, even if listener is reconfigured.
  host,_,err:=net.SplitHostPort(r.RemoteAddr)
  ip:=net.ParseIP(host)
  if err!=nil || ip==nil || !ip.IsLoopback(){http.Error(w,"forbidden",403);return}
  if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")),[]byte("Bearer "+token))!=1{
   w.Header().Set("WWW-Authenticate","Bearer")
   http.Error(w,"unauthorized",401);return
  }
  w.Header().Set("Content-Type","text/plain; version=0.0.4; charset=utf-8")
  w.Header().Set("Cache-Control","no-store")
  fmt.Fprintln(w,"# HELP ploos_video_jobs Current retained video jobs by status (not a lifetime counter).")
  fmt.Fprintln(w,"# TYPE ploos_video_jobs gauge")
  counts:=manager.MetricsSnapshot()
  for _,status:=range []string{"queued","running","completed","failed","cancelled"}{
   fmt.Fprintf(w,"ploos_video_jobs{status=%q} %d\n",status,counts[status])
  }
  // Provider metrics are an on-demand health snapshot, never a paid generation call.
  statuses:=manager.ProviderHealth(r.Context())
  fmt.Fprintln(w,"# HELP ploos_video_provider_up Provider returned HTTP 200 during the latest probe.")
  fmt.Fprintln(w,"# TYPE ploos_video_provider_up gauge")
  fmt.Fprintln(w,"# HELP ploos_video_provider_latency_seconds Duration of the latest provider health probe.")
  fmt.Fprintln(w,"# TYPE ploos_video_provider_latency_seconds gauge")
  for _,p:=range statuses{
   // Provider names originate in local configuration, not user input.
   name:=strconv.Quote(strings.ReplaceAll(p.Name,"\n",""))
   up:=0
   if p.Status=="healthy" || p.Status=="degraded"{up=1}
   fmt.Fprintf(w,"ploos_video_provider_up{provider=%s} %d\n",name,up)
   fmt.Fprintf(w,"ploos_video_provider_latency_seconds{provider=%s} %.3f\n",name,float64(p.LatencyMS)/1000)
  }
  _=time.Second
 })
}
