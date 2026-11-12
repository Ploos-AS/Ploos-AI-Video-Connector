package providerregistry

import (
 "context"
 "errors"
 "testing"
 "time"
 "github.com/Ploos-AS/Ploos-AI-Video-Connector/internal/jobs"
)
type healthFake struct{err error;delay time.Duration;calls int}
func(p *healthFake)Generate(context.Context,jobs.Request)(jobs.Result,error){panic("health check generated video")}
func(p *healthFake)Health(ctx context.Context)error{
 p.calls++
 select{
 case <-time.After(p.delay):return p.err
 case <-ctx.Done():return ctx.Err()
 }
}
func TestHealthNoGenerationAndStatus(t *testing.T){
 healthy:=&healthFake{}
 down:=&healthFake{err:errors.New("down")}
 r,err:=New([]Entry{{Name:"a",Models:[]string{"a"},Provider:healthy},{Name:"b",Models:[]string{"b"},Provider:down}})
 if err!=nil{t.Fatal(err)}
 statuses:=r.Health(context.Background())
 if len(statuses)!=2 || statuses[0].Status!="healthy" || statuses[1].Status!="unavailable"{t.Fatalf("statuses=%+v",statuses)}
 if healthy.calls!=1 || down.calls!=1{t.Fatal("health probes not executed")}
}
