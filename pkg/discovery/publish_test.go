package discovery

import "testing"

func TestPublish(t *testing.T) {
	sd := &ServiceDetail{
		Title:             "MYAPI",
		Version:           "1",
		APIName:           "myapi",
		URL:               "google.com",
		APISpec:           []byte("dummyspec"),
		Tags:              []string{"discover"},
		ServiceAttributes: nil,
	}
	agent := &Agent{}
	err := agent.publish(sd)
	if err != nil {
		t.Error(err)
	}
}
