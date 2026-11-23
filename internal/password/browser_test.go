package password

import (
 "net/http/httptest"
 "strings"
 "testing"
)
func TestSetupBoundaryAndCompletion(t *testing.T){
 origin:="http://127.0.0.1:12345";nonce:="test-nonce";done:=make(chan string,1);handler:=setupHandler(origin,nonce,done)
 send:=func(method,path,host,requestOrigin,token,body string) *httptest.ResponseRecorder {
  r:=httptest.NewRequest(method,origin+path,strings.NewReader(body));if host!=""{r.Host=host};r.Header.Set("Origin",requestOrigin);r.Header.Set("X-Pulse-Setup",token)
  w:=httptest.NewRecorder();handler.ServeHTTP(w,r);return w
 }
 if w:=send("GET","/","","","","");w.Code!=200||!strings.Contains(w.Body.String(),nonce){t.Fatal("setup HTML missing")}
 good:=`{"password":"test-password-123","confirm":"test-password-123"}`
 for _,tc:=range []struct{host,origin,token,body string;code int}{
  {"evil.test",origin,nonce,good,403},{"","https://evil.test",nonce,good,403},{"","",nonce,good,403},{"",origin,"bad",good,403},
  {"",origin,nonce,`{`,400},{"",origin,nonce,`{"password":"admin","confirm":"admin"}`,400},
  {"",origin,nonce,`{"password":"test-password-123","confirm":"different"}`,400},
 }{if w:=send("POST","/setup",tc.host,tc.origin,tc.token,tc.body);w.Code!=tc.code{t.Fatalf("got %d want %d",w.Code,tc.code)}}
 if len(done)!=0{t.Fatal("invalid request accepted")}
 if w:=send("POST","/setup","",origin,nonce,good);w.Code!=200{t.Fatal(w.Code)}
 if value:=<-done;value!="test-password-123"{t.Fatal("wrong password")}
 if w:=send("POST","/setup","",origin,nonce,good);w.Code!=409{t.Fatal("setup accepted twice")}
}
