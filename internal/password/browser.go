package password

import (
 "context"
 "crypto/rand"
 _ "embed"
 "encoding/hex"
 "encoding/json"
 "fmt"
 "log"
 "net"
 "net/http"
 "os"
 "os/exec"
 "runtime"
 "strings"
 "sync"
 "time"
)

//go:embed setup.html
var setupHTML string

// setupHandler serves only on the loopback listener. The nonce is delivered in
// same-origin HTML, never in a URL or log. Cross-site requests cannot read it.
func setupHandler(origin,nonce string,done chan<- string) http.Handler {
 var mu sync.Mutex
 accepted:=false
 return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  w.Header().Set("Cache-Control","no-store")
  w.Header().Set("X-Frame-Options","DENY")
  w.Header().Set("X-Content-Type-Options","nosniff")
  w.Header().Set("Content-Security-Policy","default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
  if r.Host!=strings.TrimPrefix(origin,"http://") || (r.Header.Get("Origin")!=""&&r.Header.Get("Origin")!=origin) {http.Error(w,"Forbidden",403);return}
  if r.Method=="GET"&&r.URL.Path=="/" {
   w.Header().Set("Content-Type","text/html; charset=utf-8")
   fmt.Fprint(w,strings.ReplaceAll(setupHTML,"SETUP_NONCE",nonce));return
  }
  if r.URL.Path!="/setup"{http.NotFound(w,r);return}
  if r.Method!="POST"{w.Header().Set("Allow","POST");http.Error(w,"Method not allowed",405);return}
  if r.Header.Get("Origin")!=origin||r.Header.Get("X-Pulse-Setup")!=nonce{http.Error(w,"Forbidden",403);return}
  var in struct{Password string `json:"password"`;Confirm string `json:"confirm"`}
  if json.NewDecoder(http.MaxBytesReader(w,r.Body,2048)).Decode(&in)!=nil {http.Error(w,"Некорректный запрос",400);return}
  if Validate(in.Password)!=nil{http.Error(w,"Пароль должен содержать от 12 до 72 байт UTF-8",400);return}
  if in.Password!=in.Confirm{http.Error(w,"Пароли не совпадают",400);return}
  mu.Lock();defer mu.Unlock()
  if accepted{http.Error(w,"Пароль уже принят",409);return}
  accepted=true
  w.Header().Set("Content-Type","application/json")
  fmt.Fprint(w,`{"accepted":true}`)
  done<-in.Password
 })
}

func prompt() (string,error) {
 ln,err:=net.Listen("tcp","127.0.0.1:0");if err!=nil{return "",err}
 defer ln.Close()
 b:=make([]byte,32);if _,err=rand.Read(b);err!=nil{return "",err}
 origin:="http://"+ln.Addr().String()
 done:=make(chan string,1)
 srv:=&http.Server{Handler:setupHandler(origin,hex.EncodeToString(b),done),ReadHeaderTimeout:5*time.Second,ReadTimeout:15*time.Second,WriteTimeout:15*time.Second,IdleTimeout:30*time.Second}
 defer srv.Close()
 failed:=make(chan error,1)
 go func(){failed<-srv.Serve(ln)}()
 log.Printf("SETUP: %s",origin)
 fmt.Printf("\nОткройте в браузере на этом компьютере: %s\nСоздайте пароль (не admin). Затем дождитесь Server listening on.\n",origin)
 if os.Getenv("PULSE_NO_BROWSER")!="1" {
  var cmd *exec.Cmd
  switch runtime.GOOS{case "windows":cmd=exec.Command("rundll32","url.dll,FileProtocolHandler",origin);case "darwin":cmd=exec.Command("open",origin);default:cmd=exec.Command("xdg-open",origin)}
  if err:=cmd.Start();err!=nil{log.Print("Не удалось открыть браузер. Откройте SETUP адрес вручную.")}else{go func(){_ = cmd.Wait()}()}
 }
 timer:=time.NewTimer(15*time.Minute);defer timer.Stop()
 select{
 case value:=<-done:
  // Wait for the success response to reach the browser before closing listener.
  ctx,cancel:=context.WithTimeout(context.Background(),5*time.Second);defer cancel()
  _=srv.Shutdown(ctx)
  return value,nil
 case err:=<-failed:return "",fmt.Errorf("setup listener failed: %w",err)
 case <-timer.C:return "",fmt.Errorf("setup timed out after 15 minutes; start Pulse Server again")
 }
}
