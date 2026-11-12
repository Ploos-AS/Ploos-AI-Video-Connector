package providerregistry

import (
 "context"
 "sort"
 "sync"
 "time"
)

type HealthStatus struct {
 Name string `json:"name"`
 Status string `json:"status"`
 LatencyMS int64 `json:"latency_ms"`
 CheckedAt time.Time `json:"checked_at"`
}
type healthProbe interface{Health(context.Context)error}

// Health performs bounded read-only checks in parallel. Degraded means slow but successful.
func(r *Registry) Health(ctx context.Context)[]HealthStatus{
 names:=make([]string,0,len(r.providers))
 for name:=range r.providers{names=append(names,name)}
 sort.Strings(names)
 results:=make([]HealthStatus,len(names))
 var wg sync.WaitGroup
 for i,name:=range names{
  wg.Add(1)
  go func(i int,name string){
   defer wg.Done()
   entry:=r.providers[name]
   result:=HealthStatus{Name:name,Status:"unavailable"}
   probe,ok:=entry.Provider.(healthProbe)
   if !ok{results[i]=result;return}
   checkCtx,cancel:=context.WithTimeout(ctx,3*time.Second)
   defer cancel()
   started:=time.Now()
   err:=probe.Health(checkCtx)
   result.LatencyMS=time.Since(started).Milliseconds()
   result.CheckedAt=time.Now().UTC()
   if err==nil{
    result.Status="healthy"
    if result.LatencyMS>=1000{result.Status="degraded"}
   }
   results[i]=result
  }(i,name)
 }
 wg.Wait()
 return results
}
