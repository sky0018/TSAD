package manager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

var (
	httpCli = &http.Client{
		Timeout: time.Second * 30,
	}
)

func leftShift30s(t time.Time) time.Time {
	unix := t.Unix()
	unix -= (unix % 30)
	return time.Unix(unix, 0)
}

// PostModel .
// if code is 200, resp will be filled,
//  else errmsg will be filled
func PostModel(url string, reqModel interface{}, respModel interface{}) error {
	buf, err := json.Marshal(reqModel)
	if err != nil {
		return fmt.Errorf("marshal err: %v", err)
	}

	reader := bytes.NewReader(buf)
	resp, err := httpCli.Post(url, "application/json", reader)
	if err != nil {
		return fmt.Errorf("post: %v, err: %v", url, err)
	}

	buf, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response err: %v", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("response err: %v", string(buf))
	}

	if respModel != nil {
		if err := json.Unmarshal(buf, respModel); err != nil {
			return fmt.Errorf("unmarshal %v err: %v", url, err)
		}
	}

	return nil
}

// GetModel same as PostModel
func GetModel(url string, respModel interface{}) error {
	resp, err := httpCli.Get(url)
	if err != nil {
		return fmt.Errorf("get: %v, err: %v", url, err)
	}

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response err: %v", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("response err: %v", string(buf))
	}

	if err := json.Unmarshal(buf, respModel); err != nil {
		return fmt.Errorf("unmarshal %v, err: %v", url, err)
	}

	return nil
}
