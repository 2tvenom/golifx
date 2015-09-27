# Go LIFX library

Pure Go Implementation of LIFX protocol. This allow you to lookup bulbs, change color, power and receive info. This code has been developed and maintained by Ven at September 2015.

![gopher](https://github.com/2tvenom/golifx/raw/master/gopher.png)

## Installation

```bash
go get github.com/2tvenom/golifx
```

## Example
```go
package main

import (
	"fmt"
	"github.com/2tvenom/golifx"
	"time"
)

func main() {
	//Lookup all bulbs
	bulbs, _ := golifx.LookupBulbs()
	//Get label
	location, _ := bulbs[0].GetLabel()

	fmt.Printf("Label: %s\n", location) //Ven LIFX

	//Get power state
	powerState, _ := bulbs[0].GetPowerState()

	//Turn if off
	if !powerState {
		bulbs[0].SetPowerState(true)
	}

	ticker := time.NewTicker(time.Second)
	counter := 0

	hsbk := &golifx.HSBK{
		Hue:        2000,
		Saturation: 13106,
		Brightness: 65535,
		Kelvin:     3200,
	}
	//Change color every second
	for _ = range ticker.C {
		bulbs[0].SetColorState(hsbk, 500)
		counter++
		hsbk.Hue += 5000
		if counter > 10 {
			ticker.Stop()
			break
		}
	}
	//Turn off
	bulbs[0].SetPowerState(false)
}


```
## Links
 - LIFX protocol specification http://lan.developer.lifx.com/
 - Community https://community.lifx.com/c/developing-with-lifx

## Licence
[WTFPL](http://www.wtfpl.net/)