package roer

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/Sirupsen/logrus"
)

func prettyPrintJSON(j []byte) {
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, j, "", "  "); err != nil {
		logrus.WithError(err).Warn("failed prettyifying response")
		fmt.Println(string(j))
		return
	}
	fmt.Println(string(pretty.Bytes()))
}
