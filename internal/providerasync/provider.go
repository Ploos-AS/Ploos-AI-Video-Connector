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
 return p.callWithKey(ctx,method,path,body,"")
}
func(p *Provider) callWithKey(ctx context.Context,method,path string,body []byte,key string)(remoteJob,error){
 req,err:=http.NewRequestWithContext(ctx,method,p.endpoint+path,bytes.NewReader(body))
 if err!=nil{return remoteJob{},err}
 req.Header.Set("Authorization","Bearer "+p.token)
 if key!="" {req.Header.Set("Idempotency-Key",key)}
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
 return p.GenerateTracked(ctx,request,func(string)error{return nil})
}
func(p *Provider) GenerateTracked(ctx context.Context,request jobs.Request,record func(string)error)(jobs.Result,error){
 return p.generate(ctx,request,"",record)
}
func(p *Provider) GenerateIdempotent(ctx context.Context,request jobs.Request,key string,record func(string)error)(jobs.Result,error){
 if len(key)!=32 {return jobs.Result{},errors.New("invalid idempotency key")}
 for _,c:=range key {if !((c>='0' && c<='9') || (c>='a' && c<='f')){return jobs.Result{},errors.New("invalid idempotency key")}}
 return p.generate(ctx,request,key,record)
}
func(p *Provider) generate(ctx context.Context,request jobs.Request,key string,record func(string)error)(jobs.Result,error){
 body,err:=json.Marshal(request)
 if err!=nil{return jobs.Result{},err}
 remote,err:=p.callWithKey(ctx,http.MethodPost,"",body,key)
 if err!=nil{return jobs.Result{},err}
 // If persistence fails, cancel the remote job instead of risking an untracked charge.
 if strings.ContainsAny(remote.ID,"/?#") {return jobs.Result{},errors.New("unsafe remote job ID")}
 if err:=record(remote.ID);err!=nil{
  cancelCtx,cancel:=context.WithTimeout(context.Background(),2*time.Second)
  defer cancel()
  _,_=p.call(cancelCtx,http.MethodDelete,"/"+url.PathEscape(remote.ID),nil)
  return jobs.Result{},err
 }
 return p.poll(ctx,remote)
}
func(p *Provider) Resume(ctx context.Context,remoteID string)(jobs.Result,error){
 if remoteID=="" || len(remoteID)>128 || strings.ContainsAny(remoteID,"/?#"){return jobs.Result{},errors.New("invalid remote job ID")}
 remote,err:=p.call(ctx,http.MethodGet,"/"+url.PathEscape(remoteID),nil)
 if err!=nil{return jobs.Result{},err}
 if remote.ID!=remoteID{return jobs.Result{},errors.New("remote job ID mismatch")}
 return p.poll(ctx,remote)
}
func(p *Provider) poll(ctx context.Context,remote remoteJob)(jobs.Result,error){
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
   next,err:=p.call(ctx,http.MethodGet,"/"+url.PathEscape(remote.ID),nil)
   if err!=nil{return jobs.Result{},err}
   if next.ID!=remote.ID{return jobs.Result{},errors.New("remote job ID changed")}
   remote=next
  }
 }
}
