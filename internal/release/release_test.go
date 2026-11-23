package release

import (
 "context"
 "testing"
)

func TestAvailable(t *testing.T) {
 old:=Version;defer func(){Version=old}()
 for _,tc:=range []struct{current,latest string;want bool}{{"dev","v1.0.0",false},{"v1.0.0","v1.0.0",false},{"v1.0.0","v1.1.0",true},{"v1.0.0","",false}}{
 Version=tc.current;if got:=Available(Release{Tag:tc.latest});got!=tc.want{t.Errorf("%s -> %s: %v",tc.current,tc.latest,got)}
 }
}
func TestRejectUntrustedURL(t *testing.T){for _,u:=range []string{"http://github.com/file","https://example.com/file","https://github.com.evil.test/file"}{if _,err:=get(context.Background(),u,100);err==nil{t.Fatalf("accepted %s",u)}}}
