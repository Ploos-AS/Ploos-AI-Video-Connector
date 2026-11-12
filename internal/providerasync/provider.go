// Package providerasync implements a private, asynchronous HTTP video provider.
package providerasync

import (
 "bytes"
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "net"
 "net/http"
 "net/netip"
 "net/url"
 "strings"
 "time"

 "github.com/Ploos-AS/Ploos-AI-Video-Connector/internal/jobs"
)

type Provider struct {
 endpoint string
 token string
 client *http.Client
 interval time.Duration
}

type remoteJob struct {
 ID string `json:"id"`
 Status string `json:"status"`
 URL string `json:"url,omitempty"`
 MIMEType string `json:"mime_type,omitempty"`
}

func New(endpoint,token string,interval time.Duration)(*Provider,error){
 u,err:=url.Parse(endpoint)
 if err!=nil || u.Scheme!="http" || u.User!=nil || u.RawQuery!="" || u.Fragment!="" || u.Path!="/jobs" || u.Host==""{return nil,errors.New("invalid async provider endpoint")}
 host,port,err:=net.SplitHostPort(u.Host)
 if err!=nil || port==""{return nil,errors.New("explicit IP and port required")}
 ip,err:=netip.ParseAddr(host)
 if err!=nil || ip.Is4In6() || ip.Zone()!="" || !(ip.IsPrivate() || ip.IsLoopback()){return nil,errors.New("private IP required")}
 if strings.TrimSpace(token)=="" || interval<100*time.Millisecond{return nil,errors.New("invalid async provider configuration")}
 transport:=&http.Transport{Proxy:nil,DisableKeepAlives:true,DialContext:(&net.Dialer{Timeout:3*time.Second}).DialContext}
 client:=&http.Client{Transport:transport,Timeout:10*time.Second,CheckRedirect:func(*http.Request,[]*http.Request)error{return http.ErrUseLastResponse}}
 return &Provider{endpoint:endpoint,token:token,client:client,interval:interval},nil
}

func(p *Provider) call(ctx context.Context,method,path string,body []byte)(remoteJob,error){
 req,err:=http.NewRequestWithContext(ctx,method,p.endpoint+path,bytes.NewReader(body))
 if err!=nil{return remoteJob{},err}
 req.Header.Set("Authorization","Bearer "+p.token)
 req.Header.Set("Content-Type","application/json")
 res,err:=p.client.Do(req)
 if err!=nil{return remoteJob{},err}
 defer res.Body.Close()
 if res.StatusCode!=http.StatusOK && res.StatusCode!=http.StatusAccepted{return remoteJob{},fmt.Errorf("provider returned HTTP %d",res.StatusCode)}
 data,err:=io.ReadAll(io.LimitReader(res.Body,16385))
 if err!=nil || len(data)>16384{return remoteJob{},errors.New("invalid provider response size")}
 var job remoteJob
 if json.Unmarshal(data,&job)!=nil || job.ID=="" || len(job.ID)>128{return remoteJob{},errors.New("invalid provider job")}
 return job,nil
}

func(p *Provider) Generate(ctx context.Context,request jobs.Request)(jobs.Result,error){
 body,err:=json.Marshal(request)
 if err!=nil{return jobs.Result{},err}
 remote,err:=p.call(ctx,http.MethodPost,"",body)
 if err!=nil{return jobs.Result{},err}
 // Cancellation is best-effort: do not block the caller on an unreachable provider.
 defer func(){
  if ctx.Err()==nil{return}
  cancelCtx,cancel:=context.WithTimeout(context.Background(),2*time.Second)
  defer cancel()
  _,_=p.call(cancelCtx,http.MethodDelete,"/"+url.PathEscape(remote.ID),nil)
 }()
 ticker:=time.NewTicker(p.interval)
 defer ticker.Stop()
 for{
  switch remote.Status{
  case "completed":
   parsed,err:=url.Parse(remote.URL)
   if err!=nil || parsed.Scheme!="https" || parsed.Host=="" || parsed.User!=nil || parsed.Fragment!="" || remote.MIMEType!="video/mp4"{return jobs.Result{},errors.New("invalid completed video result")}
   return jobs.Result{URL:remote.URL,MIMEType:remote.MIMEType},nil
  case "failed","cancelled":return jobs.Result{},errors.New("remote video job failed")
  case "queued","running","in_progress":
  default:return jobs.Result{},errors.New("unknown remote video status")
  }
  select{
  case <-ctx.Done():return jobs.Result{},ctx.Err()
  case <-ticker.C:
   remote,err=p.call(ctx,http.MethodGet,"/"+url.PathEscape(remote.ID),nil)
   if err!=nil{return jobs.Result{},err}
  }
 }
}
