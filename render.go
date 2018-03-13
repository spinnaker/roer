package roer

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
)

func prettyPrintJSON(j []byte) {
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, j, "", "  "); err != nil {
		logrus.WithError(err).Warn("failed prettyifying response")
		logrus.Error(string(j))
		return
	}
	fmt.Println(string(pretty.Bytes()))
}
