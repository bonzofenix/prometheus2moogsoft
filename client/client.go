package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Severity int

const (
	CLEAR Severity = iota
	INDETERMINATE
	MINOR
	MAJOR
	CRITICAL
)

func (s Severity) String() string {
	return [...]string{"CLEAR", "INDETERMINATE", "MINOR", "MAJOR", "CRITICAL"}[s]
}

// Moogsoft client
type Client struct {
	Env               string
	URL               string
	EventsEndpoint    string
	XMattersGroupName string
}

// INPUT
type PrometheusPayload struct {
	Alerts []PrometheusAlert `json:"alerts"`
}

// INPUT
type PrometheusAlert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     string            `json:"startsAt"`
	GeneratorURL string            `json:"generatorURL"`
}

func (a PrometheusAlert) GetSeverity() Severity {
	alertStatus := fmt.Sprintf("%s-%s", a.Status, a.Labels["severity"])

	switch alertStatus {
	case "firing-warning":
		return MAJOR
	case "firing-critical":
		return CRITICAL
	case "resolved-warning":
		return CLEAR
	case "resolved-critical":
		return CLEAR
	default:
		return INDETERMINATE
	}
}

func (a PrometheusAlert) GetAgentTime() string {
	agentTime, err := time.Parse(time.RFC3339Nano, a.StartsAt)
	if err != nil {
		log.Println(err.Error())
	}

	return strconv.FormatInt(agentTime.Unix(), 10)
}

//OUTPUT
type MoogsoftPayload struct {
	Events []MoogsoftEvent `json:"events"`
}

//OUTPUT
type MoogsoftEvent struct {
	Signature              string   `json:"signature"`
	Source                 string   `json:"source"`
	SourceId               string   `json:"source_id"`
	ExternalId             string   `json:"external_id"`
	Manager                string   `json:"manager"`
	Class                  string   `json:"class"`
	AgentLocation          string   `json:"agent_location"`
	Type                   string   `json:"type"`
	Severity               Severity `json:"severity"`
	Description            string   `json:"description"`
	AgentTime              string   `json:"agent_time"`
	Agent                  string   `json:"agent"`
	AonMetricName          string   `json:"aonMetricName"`
	AonMetricValue         string   `json:"aonMetricValue"`
	AonMonitoredEntityName string   `json:"aonMonitoredEntityName"`
	AonXMattersGroupName   string   `json:"aonXMattersGroupName"`
	AonSNOWGroupName       string   `json:"aonSNOWGroupName"` //empty
	AonToolUrl             string   `json:"aonToolURL"`       // prometheus url
	AonIPAddress           string   `json:"aonIPAddress"`
	AonIPSubnet            string   `json:"aonIPSubnet"`
	AonJSONVersion         string   `json:"aonJSONversion"`
}

func (c *Client) SendEvents(payload string, token string) (int, error) {
	var moogsoftEvents []MoogsoftEvent
	var prometheusPayload PrometheusPayload
	if os.Getenv("DEBUG") != "" {
		log.Println("Received payload: ", payload)
	}

	err := json.Unmarshal([]byte(payload), &prometheusPayload)
	if err != nil {
		return 500, err
	}

	for _, alert := range prometheusPayload.Alerts {
		event, err := c.eventFor(alert)
		if err != nil {
			log.Println(err.Error())
		}

		moogsoftEvents = append(moogsoftEvents, event)
	}

	moogsoftPayload := MoogsoftPayload{
		Events: moogsoftEvents,
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

func (c *Client) eventFor(alert PrometheusAlert) (MoogsoftEvent, error) {
	moogsoftEvent := MoogsoftEvent{
		Type:                 alert.Labels["service"],
		Description:          alert.Annotations["description"],
		AonToolUrl:           alert.GeneratorURL,
		AonXMattersGroupName: c.XMattersGroupName,
		Manager:              "Prometheus",
		Class:                "PCF",
		Severity:             alert.GetSeverity(),
		AonJSONVersion:       "2",
		Agent:                c.Env,
		AgentTime:            alert.GetAgentTime(),
	}

	var err error

	switch moogsoftEvent.Type {
	case "bosh-deployment", "bosh-job", "bosh-job-process":
		moogsoftEvent.Signature = fmt.Sprintf("%s::%s::%s::%s::%s::%s::%s", alert.Labels["alertname"], alert.Labels["environment"], alert.Labels["bosh_name"], alert.Labels["bosh_job_az"], alert.Labels["bosh_deployment"], alert.Labels["bosh_job_name"], alert.Labels["bosh_job_index"])
		moogsoftEvent.SourceId = ""
		moogsoftEvent.AonIPAddress = alert.Labels["bosh_job_ip"]

	case "prometheus":
		moogsoftEvent.Signature = fmt.Sprintf("%s::%s::%s", alert.Labels["alertname"], alert.Labels["bosh_deployment"], alert.Labels["job"])

	case "cf":
		moogsoftEvent.Signature = fmt.Sprintf("%s::%s::%s", alert.Labels["alertname"], alert.Labels["environment"], alert.Labels["bosh_deployment"])

	default:
		err = errors.New(fmt.Sprintf("Unsopported service: %s", moogsoftEvent.Type))
		moogsoftEvent.Signature = alert.Annotations["description"]
		moogsoftEvent.Severity = 1
	}

	moogsoftEvent.ExternalId = moogsoftEvent.Signature

	return moogsoftEvent, err
}
