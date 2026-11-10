package main

import (
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "net/http"
 "net/url"
 "strings"
 "time"
)

// introspector validates opaque OAuth access tokens with a trusted RFC 7662 endpoint.
// It fails closed if any required issuer, audience, scope or expiration is missing.
type introspector struct {
 endpoint, clientID, clientSecret, issuer, audience, scope string
 client *http.Client
}
func newIntrospector(endpoint,id,secret,issuer,audience,scope string) (*introspector,error) {
 if !validHTTPS(issuer) || !validHTTPS(audience) {return nil,errors.New("issuer and audience must be HTTPS origins")}
 u,err:=url.Parse(endpoint)
 if err!=nil || u.Scheme!="https" || u.Host=="" || u.User!=nil || u.Fragment!="" || u.RawQuery!="" {return nil,errors.New("introspection endpoint must be HTTPS without query, fragment or userinfo")}
 if id=="" || secret=="" || scope=="" {return nil,errors.New("introspection credentials and required scope must be configured")}
 return &introspector{endpoint: endpoint,clientID:id,clientSecret:secret,issuer:issuer,audience:audience,scope:scope,
  client:&http.Client{Timeout:5*time.Second,CheckRedirect:func(*http.Request,[]*http.Request)error{return http.ErrUseLastResponse}}},nil
}
func (i *introspector) validate(ctx context.Context, token string) error {
 if token=="" || len(token)>8192 {return errors.New("invalid token")}
 form:=url.Values{"token":{token},"token_type_hint":{"access_token"}}
 req,err:=http.NewRequestWithContext(ctx,http.MethodPost,i.endpoint,strings.NewReader(form.Encode()))
 if err!=nil{return err}
 req.Header.Set("Content-Type","application/x-www-form-urlencoded")
 req.SetBasicAuth(i.clientID,i.clientSecret)
 res,err:=i.client.Do(req)
 if err!=nil{return err}
 defer res.Body.Close()
 if res.StatusCode!=200 {return fmt.Errorf("introspection failed: %d",res.StatusCode)}
 var claims struct{
  Active bool `json:"active"`
  Iss string `json:"iss"`
  Aud json.RawMessage `json:"aud"`
  Exp int64 `json:"exp"`
  Scope string `json:"scope"`
 }
 dec:=json.NewDecoder(io.LimitReader(res.Body,16384))
 if err:=dec.Decode(&claims);err!=nil{return err}
 if !claims.Active || claims.Iss!=i.issuer || claims.Exp<=time.Now().Unix(){return errors.New("inactive, expired or wrong issuer")}
 var aud string
 if json.Unmarshal(claims.Aud,&aud)==nil {
  if aud!=i.audience{return errors.New("wrong audience")}
 }else{
  var audiences []string
  if json.Unmarshal(claims.Aud,&audiences)!=nil{return errors.New("missing audience")}
  found:=false
  for _,a:=range audiences{if a==i.audience{found=true}}
  if !found{return errors.New("wrong audience")}
 }
 for _,s:=range strings.Fields(claims.Scope){if s==i.scope{return nil}}
 return errors.New("missing required scope")
}
