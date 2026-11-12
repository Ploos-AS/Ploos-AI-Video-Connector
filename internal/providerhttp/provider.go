package providerhttp

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

const maxResponse = 16 << 10

type Provider struct {
 endpoint string
 token string
 client *http.Client
}

func New(endpoint, token string) (*Provider,error) {
 u,err:=url.Parse(endpoint)
 if err!=nil || u.Scheme!="http" || u.User!=nil || u.RawQuery!="" || u.Fragment!="" || u.Path!="/generate" || u.Host=="" {return nil,errors.New("invalid video provider endpoint")}
 host,port,err:=net.SplitHostPort(u.Host)
 if err!=nil || port=="" {return nil,errors.New("video provider requires explicit IP and port")}
 ip,err:=netip.ParseAddr(host)
 if err!=nil || !(ip.IsPrivate() || ip.IsLoopback()) || ip.Is4In6() {return nil,errors.New("video provider must use private IP")}
 if strings.TrimSpace(token)=="" {return nil,errors.New("video provider token required")}
 transport:=&http.Transport{Proxy:nil,DisableKeepAlives:true,DialContext:(&net.Dialer{Timeout:3*time.Second}).DialContext}
 client:=&http.Client{Transport:transport,Timeout:2*time.Minute,CheckRedirect:func(*http.Request,[]*http.Request)error{return http.ErrUseLastResponse}}
 return &Provider{endpoint:endpoint,token:token,client:client},nil
}

func (p *Provider) Generate(ctx context.Context,request jobs.Request)(jobs.Result,error){
 body,err:=json.Marshal(request)
 if err!=nil{return jobs.Result{},err}
 req,err:=http.NewRequestWithContext(ctx,http.MethodPost,p.endpoint,bytes.NewReader(body))
 if err!=nil{return jobs.Result{},err}
 req.Header.Set("Content-Type","application/json")
 req.Header.Set("Authorization","Bearer "+p.token)
 res,err:=p.client.Do(req)
 if err!=nil{return jobs.Result{},fmt.Errorf("video provider request failed: %w",err)}
 defer res.Body.Close()
 if res.StatusCode!=http.StatusOK{return jobs.Result{},errors.New("video provider rejected request")}
 data,err:=io.ReadAll(io.LimitReader(res.Body,maxResponse+1))
 if err!=nil || len(data)>maxResponse{return jobs.Result{},errors.New("invalid video provider response")}
 var output jobs.Result
 if err:=json.Unmarshal(data,&output);err!=nil{return jobs.Result{},errors.New("invalid video provider JSON")}
 u,err:=url.Parse(output.URL)
 if err!=nil || u.Scheme!="https" || u.Host=="" || u.User!=nil || u.Fragment!="" || output.MIMEType!="video/mp4"{return jobs.Result{},errors.New("invalid video provider result")}
 return output,nil
}
