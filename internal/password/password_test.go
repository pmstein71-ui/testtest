package password

import (
 "strings"
 "testing"
)
func TestValidate(t *testing.T){
 for _,tc:=range []struct{value string;valid bool}{{"",false},{strings.Repeat("x",11),false},{strings.Repeat("x",12),true},{strings.Repeat("x",72),true},{strings.Repeat("x",73),false},{strings.Repeat("я",36),true},{strings.Repeat("я",37),false}} {
  if got:=Validate(tc.value)==nil;got!=tc.valid{t.Errorf("length %d: valid=%v",len(tc.value),got)}
 }
}
func TestEnvironment(t *testing.T){t.Setenv("ADMIN_PASSWORD","test-password-123");t.Setenv("PULSE_NO_DIALOG","1");value,err:=Read();if err!=nil||value!="test-password-123"{t.Fatal("env bootstrap failed")}}
func TestHeadlessMissingPassword(t *testing.T){t.Setenv("ADMIN_PASSWORD","");t.Setenv("PULSE_NO_DIALOG","1");if _,err:=Read();err==nil{t.Fatal("expected actionable error, not a dialog")}}
