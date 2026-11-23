package release

import (
 "context"
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "fmt"
 "io"
 "net/http"
 "net/url"
 "os"
 "path/filepath"
 "strings"
 "time"
)
var Version="dev"
const Repo="pmstein71-ui/testtest"
type Asset struct{Name string `json:"name"`;URL string `json:"browser_download_url"`}
type Release struct{Tag string `json:"tag_name"`;Assets []Asset `json:"assets"`}
var client=&http.Client{Timeout:5*time.Minute}
func get(ctx context.Context,u string,max int64)([]byte,error){p,e:=url.Parse(u);if e!=nil||p.Scheme!="https"||(p.Host!="api.github.com"&&p.Host!="github.com"){return nil,fmt.Errorf("untrusted release URL")};req,e:=http.NewRequestWithContext(ctx,"GET",u,nil);if e!=nil{return nil,e};req.Header.Set("User-Agent","Pulse/"+Version);r,e:=client.Do(req);if e!=nil{return nil,e};defer r.Body.Close();if r.StatusCode!=200{return nil,fmt.Errorf("GitHub HTTP %d",r.StatusCode)};b,e:=io.ReadAll(io.LimitReader(r.Body,max+1));if int64(len(b))>max{return nil,fmt.Errorf("download too large")};return b,e}
func Latest(ctx context.Context)(Release,error){var r Release;b,e:=get(ctx,"https://api.github.com/repos/"+Repo+"/releases/latest",1<<20);if e!=nil{return r,e};e=json.Unmarshal(b,&r);return r,e}
func Available(r Release)bool{return Version!="dev"&&r.Tag!=""&&r.Tag!=Version}
func Download(ctx context.Context)(string,error){r,e:=Latest(ctx);if e!=nil{return "",e};if !Available(r){return "",fmt.Errorf("no update available")};var installer,checks string;for _,a:=range r.Assets{if a.Name=="setup.exe"{installer=a.URL};if a.Name=="checksums.txt"{checks=a.URL}};if installer==""||checks==""{return "",fmt.Errorf("release assets missing")};manifest,e:=get(ctx,checks,1<<20);if e!=nil{return "",e};data,e:=get(ctx,installer,200<<20);if e!=nil{return "",e};sum:=sha256.Sum256(data);valid:=false;for _,line:=range strings.Split(string(manifest),"\n"){f:=strings.Fields(line);if len(f)==2&&strings.TrimPrefix(f[1],"*")=="setup.exe"&&strings.EqualFold(f[0],hex.EncodeToString(sum[:])){valid=true}};if !valid{return "",fmt.Errorf("installer checksum mismatch")};dir,e:=os.MkdirTemp("","pulse-update-");if e!=nil{return "",e};path:=filepath.Join(dir,"setup.exe");e=os.WriteFile(path,data,0700);return path,e}
