package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// Moogsoft client
type Client struct {
	URL            string
	EventsEndpoint string
}

type PrometheusPayload struct {
	Alerts []PrometheusAlert `json:"alerts"`
}

type PrometheusAlert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

type MoogsoftEvent struct {
	Signature              string `json:"signature"`
	Source                 string `json:"source"`
	SourceId               string `json:"source_id"`
	ExternalId             string `json:"external_id"`
	Manager                string `json:"manager"`
	Class                  string `json:"class"`
	AgentLocation          string `json:"agent_location"`
	Type                   string `json:"type"`
	Severity               int    `json:"severity"`
	Description            string `json:"description"`
	AgentTime              string `json:"agent_time"`
	Agent                  string `json:"agent"`
	AonMetricName          string `json:"aonMetricName"`
	AonMetricValue         string `json:"aonMetricValue"`
	AonMonitoredEntityName string `json:"aonMonitoredEntityName"`
	AonXMattersGroupName   string `json:"aonXMattersGroupName"`
	AonSNOWGroupName       string `json:"aonSNOWGroupName"`
	AonToolUrl             string `json:"aonToolURL"`
	AonIPAddress           string `json:"aonIPAddress"`
	AonIPSubnet            string `json:"aonIPSubnet"`
	AonJSONVersion         string `json:"aonJSONversion"`
}

type MoogsoftPayload struct {
	Events []MoogsoftEvent `json:"events"`
}

func (c *Client) SendEvents(payload string, token string) (int, error) {
	var prometheusPayload PrometheusPayload
	err := json.Unmarshal([]byte(payload), &prometheusPayload)
	if err != nil {
		return 500, err
	}

	moogsoftEvent := MoogsoftEvent{}

	alert := prometheusPayload.Alerts[0]
	switch service := alert.Labels["service"]; service {
	case "bosh-job":
		moogsoftEvent.Signature = fmt.Sprintf("%s::%s/%s/%s/%s/%s/%s", alert.Labels["alertname"], alert.Labels["environment"], alert.Labels["bosh_name"], alert.Labels["bosh_job_az"], alert.Labels["bosh_deployment"], alert.Labels["bosh_job_name"], alert.Labels["bosh_job_index"])
		moogsoftEvent.AgentTime = "1540313079"
		moogsoftEvent.SourceId = fmt.Sprintf("%s-%s", moogsoftEvent.AgentTime, moogsoftEvent.Signature)
		moogsoftEvent.ExternalId = fmt.Sprintf("%s/%s/%s", alert.Labels["environment"], alert.Labels["bosh_name"], alert.Labels["bosh_deployment"])
		moogsoftEvent.Manager = alert.Labels["bosh_job_id"]
		moogsoftEvent.Class = fmt.Sprintf("%s/%s", alert.Labels["environment"], alert.Labels["bosh_job_az"])
		moogsoftEvent.Type = alert.Labels["alertname"]
		moogsoftEvent.Severity = 4
		moogsoftEvent.Description = alert.Annotations["description"]
		moogsoftEvent.AonXMattersGroupName = "xmatter-group-id"
		moogsoftEvent.AonToolUrl = "https://prometheus.your-domain.com/graph?g0.expr=up+%3D%3D+0\u0026g0.tab=1"
		moogsoftEvent.AonJSONVersion = "2"

	case "prometheus":
		moogsoftEvent.Signature = fmt.Sprintf("%s::%s/%s", alert.Labels["alertname"], alert.Labels["bosh_deployment"], alert.Labels["job"])

	}

	moogsoftPayload := MoogsoftPayload{
		Events: []MoogsoftEvent{moogsoftEvent},
	}
	rawData, err := json.Marshal(moogsoftPayload)
	if err != nil {
		return 500, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s%s", c.URL, c.EventsEndpoint), bytes.NewReader(rawData))
	if err != nil {
		return 500, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Basic %s", token))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 500, err
	}

	return res.StatusCode, err
}
