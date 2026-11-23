// Package password collects the bootstrap password without exposing it in logs.
package password

import (
 "fmt"
 "os"
)

func Validate(value string) error {
 if len(value)<12 || len(value)>72 { return fmt.Errorf("password must contain 12-72 UTF-8 bytes") }
 return nil
}

func Read() (string,error) {
 if value:=os.Getenv("ADMIN_PASSWORD");value!="" {return value,Validate(value)}
 if os.Getenv("PULSE_NO_DIALOG")=="1" {return "",fmt.Errorf("first start requires ADMIN_PASSWORD when PULSE_NO_DIALOG=1")}
 return prompt()
}
