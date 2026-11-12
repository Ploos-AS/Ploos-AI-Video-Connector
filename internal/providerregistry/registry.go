// Package providerregistry routes explicitly configured model IDs to providers.
// There is no implicit fallback, particularly not to paid providers.
package providerregistry

import (
 "context"
 "errors"
 "sort"
 "strings"

 "github.com/Ploos-AS/Ploos-AI-Video-Connector/internal/jobs"
)

var ErrConfiguration=errors.New("invalid provider registry configuration")

type Entry struct{
 Name string `json:"name"`
 Models []string `json:"models"`
 Local bool `json:"local"`
 Provider jobs.Provider `json:"-"`
}

type Model struct{
 ID string `json:"id"`
 Provider string `json:"provider"`
 Local bool `json:"local"`
}

type Registry struct{
 providers map[string]Entry
 routes map[string]string
 models []Model
}

func New(entries []Entry)(*Registry,error){
 r:=&Registry{providers:make(map[string]Entry),routes:make(map[string]string)}
 for _,e:=range entries{
  if e.Name=="" || strings.TrimSpace(e.Name)!=e.Name || e.Provider==nil || len(e.Models)==0{return nil,ErrConfiguration}
  if _,exists:=r.providers[e.Name];exists{return nil,ErrConfiguration}
  r.providers[e.Name]=e
  for _,model:=range e.Models{
   if model=="" || strings.TrimSpace(model)!=model || len(model)>128{return nil,ErrConfiguration}
   if _,exists:=r.routes[model];exists{return nil,ErrConfiguration}
   r.routes[model]=e.Name
   r.models=append(r.models,Model{ID:model,Provider:e.Name,Local:e.Local})
  }
 }
 sort.Slice(r.models,func(i,j int)bool{return r.models[i].ID<r.models[j].ID})
 return r,nil
}
func(r *Registry)Models()[]Model{return append([]Model(nil),r.models...)}
func(r *Registry)ProviderFor(model string)(jobs.Provider,error){
 name,ok:=r.routes[model]
 if !ok{return nil,jobs.ErrUnavailable}
 return r.providers[name].Provider,nil
}
func(r *Registry)Generate(ctx context.Context,req jobs.Request)(jobs.Result,error){
 p,err:=r.ProviderFor(req.Model)
 if err!=nil{return jobs.Result{},err}
 return p.Generate(ctx,req)
}
// GenerateTracked retains the remote ID callback for recovery-aware providers.
func(r *Registry)GenerateTracked(ctx context.Context,req jobs.Request,record func(string)error)(jobs.Result,error){
 p,err:=r.ProviderFor(req.Model)
 if err!=nil{return jobs.Result{},err}
 tracked,ok:=p.(jobs.RecoverableProvider)
 if !ok{return p.Generate(ctx,req)}
 return tracked.GenerateTracked(ctx,req,record)
}
func(r *Registry)GenerateIdempotent(ctx context.Context,req jobs.Request,key string,record func(string)error)(jobs.Result,error){
 p,err:=r.ProviderFor(req.Model)
 if err!=nil{return jobs.Result{},err}
 if idempotent,ok:=p.(jobs.IdempotentProvider);ok{return idempotent.GenerateIdempotent(ctx,req,key,record)}
 if tracked,ok:=p.(jobs.RecoverableProvider);ok{return tracked.GenerateTracked(ctx,req,record)}
 return p.Generate(ctx,req)
}
// ResumeFor uses the persisted model route, never another provider.
func(r *Registry)ResumeFor(ctx context.Context,model,remoteID string)(jobs.Result,error){
 p,err:=r.ProviderFor(model)
 if err!=nil{return jobs.Result{},err}
 recoverable,ok:=p.(jobs.RecoverableProvider)
 if !ok{return jobs.Result{},jobs.ErrUnavailable}
 return recoverable.Resume(ctx,remoteID)
}
