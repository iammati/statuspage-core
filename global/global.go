package global

import (
	"fmt"
	"time"
)

var BOOTUP_TIME = time.Now()
var LOGFILENAME = fmt.Sprintf("logs/%s.log", BOOTUP_TIME.Format(time.RFC3339))
