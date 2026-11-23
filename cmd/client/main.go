package main
import (
 "encoding/json"
 "fmt"
 "log"
 "net"
 "net/http"
 "net/http/httputil"
 "net/url"
 "os"
 "os/exec"
 "runtime"
 "strings"
 "time"
 "github.com/pmstein71-ui/testtest/internal/release"
 "github.com/pmstein71-ui/testtest/internal/web"
 "github.com/pmstein71-ui/testtest/internal/startup"
)
func main(){startup.Run("client",run)}
func run() error {
 server:=os.Getenv("SERVER_URL");if server==""{server="http://127.0.0.1:8080"};u,err:=url.Parse(server);if err!=nil||u.Host==""{return fmt.Errorf("invalid SERVER_URL: use an absolute http(s) URL")};if u.Scheme!="https"&&(u.Scheme!="http"||(u.Hostname()!="127.0.0.1"&&u.Hostname()!="localhost"&&u.Hostname()!="::1")){return fmt.Errorf("remote server requires HTTPS")}
 log.Printf("Connecting to server %s://%s",u.Scheme,u.Host)
 ln,err:=net.Listen("tcp","127.0.0.1:0");if err!=nil{return err};origin:="http://"+ln.Addr().String();mux:=http.NewServeMux();proxy:=httputil.NewSingleHostReverseProxy(u);proxy.ErrorHandler=func(w http.ResponseWriter,r *http.Request,e error){log.Printf("Server unavailable: %v",e);http.Error(w,"Сервер недоступен. Запустите Pulse Server и проверьте SERVER_URL. Журнал: Pulse/logs/server.log",http.StatusBadGateway)};proxy.Transport=&http.Transport{ResponseHeaderTimeout:20*time.Second};mux.Handle("/api/",proxy)
 mux.HandleFunc("GET /local/update",func(w http.ResponseWriter,r *http.Request){ctx:=r.Context();latest,e:=release.Latest(ctx);if e!=nil{http.Error(w,e.Error(),502);return};w.Header().Set("Content-Type","application/json");json.NewEncoder(w).Encode(map[string]any{"current":release.Version,"latest":latest.Tag,"available":release.Available(latest)})})
 mux.HandleFunc("POST /local/install",func(w http.ResponseWriter,r *http.Request){if runtime.GOOS!="windows"{http.Error(w,"Windows only",400);return};path,e:=release.Download(r.Context());if e==nil{e=exec.Command(path).Start()};if e!=nil{http.Error(w,e.Error(),500);return};w.Header().Set("Content-Type","application/json");fmt.Fprint(w,`{"started":true}`)})
 mux.HandleFunc("GET /",func(w http.ResponseWriter,r *http.Request){if r.URL.Path!="/"{http.NotFound(w,r);return};w.Header().Set("Content-Type","text/html; charset=utf-8");fmt.Fprint(w,web.HTML)})
 handler:=http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){w.Header().Set("Cache-Control","no-store");w.Header().Set("X-Frame-Options","DENY");w.Header().Set("X-Content-Type-Options","nosniff");if r.Host!=ln.Addr().String()|| (r.Header.Get("Origin")!=""&&r.Header.Get("Origin")!=origin){http.Error(w,"Forbidden",403);return};if (strings.HasPrefix(r.URL.Path,"/api/")||strings.HasPrefix(r.URL.Path,"/local/"))&&r.Header.Get("X-Pulse-Client")!="1"{http.Error(w,"Forbidden",403);return};mux.ServeHTTP(w,r)})
 log.Println("Open",origin)
 if runtime.GOOS=="windows"&&os.Getenv("PULSE_NO_BROWSER")!="1"{if err:=exec.Command("rundll32","url.dll,FileProtocolHandler",origin).Start();err!=nil{log.Printf("Cannot open browser: %v; open %s manually",err,origin)}}
 return (&http.Server{Handler:handler,ReadHeaderTimeout:5*time.Second,IdleTimeout:60*time.Second}).Serve(ln)
}
