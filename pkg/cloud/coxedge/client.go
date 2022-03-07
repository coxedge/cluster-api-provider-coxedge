package coxedge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"moul.io/http2curl"
)

const (
	baseUrl = "https://portal.coxedge.com/api/v1/"
)

var (
	ErrWorkloadNotFound = errors.New("workload not found")
)

type Client struct {
	client      *http.Client
	apiKey      string
	baseUrl     *url.URL
	debug       bool
	service     string
	environment string
}

func NewClient(service, environment, apiKey string, httpClient *http.Client) (*Client, error) {

	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	url, err := url.Parse(baseUrl)
	if err != nil {
		return nil, err
	}
	client := &Client{
		apiKey:  apiKey,
		baseUrl: url,
		client:  httpClient,
	}

	client.service = service
	client.environment = environment
	client.debug = os.Getenv("COX_DEBUG") != ""

	return client, nil
}

// curl -X 'GET' -H 'Mc-Api-Key: $TOKEN' 'https://portal.coxedge.com/api/v1/services/edge-services/faefawef/workloads/549ec584-c62b-4647-9ca0-f04f9a88403d'
func (c *Client) GetWorkload(id string) (*Workload, *ErrorResponse, error) {
	w := &Workload{}
	resp, err := c.DoRequest("GET", fmt.Sprintf("/services/%s/%s/workloads/%s", c.service, c.environment, id), nil, w)
	if err != nil {
		return nil, resp, err
	}
	return w, resp, nil
}

func (c *Client) GetWorkloadByName(name string) (*WorkloadData, error) {
	workloads, _, err := c.GetWorkloads()
	if err != nil {
		return nil, err
	}
	name = shortenName(name, 18)

	for _, workload := range workloads.Data {
		if workload.Name == name {
			return &workload, nil
		}
	}
	return nil, ErrWorkloadNotFound
}

// curl -X 'GET' -H 'Mc-Api-Key: $TOKEN' 'https://portal.coxedge.com/api/v1/services/edge-services/faefawef/workloads'
func (c *Client) GetWorkloads() (*Workloads, *ErrorResponse, error) {
	w := &Workloads{}
	resp, err := c.DoRequest("GET", fmt.Sprintf("/services/%s/%s/workloads", c.service, c.environment), nil, w)
	if err != nil {
		return nil, resp, err
	}
	return w, resp, nil
}

// curl -X 'GET' -H 'Mc-Api-Key: $TOKEN' 'https://portal.coxedge.com/api/v1/services/edge-services/faefawef/instances?workloadId=5e1eb085-e9b3-447b-8a0e-c0147fc0ea4d' | jq
func (c *Client) GetInstances(workloadID string) (*Instances, *ErrorResponse, error) {
	i := &Instances{}
	resp, err := c.DoRequest("GET", fmt.Sprintf("/services/%s/%s/instances?workloadId=%s", c.service, c.environment, workloadID), nil, i)
	if err != nil {
		return nil, resp, err
	}
	return i, resp, nil
}

// curl -X 'GET' -H 'Mc-Api-Key: $TOKEN' 'https://portal.coxedge.com/api/v1/services/edge-services/faefawef/instances/5e1eb085-e9b3-447b-8a0e-c0147fc0ea4d/capi-test-jg90-wi-peter-qhl-waw-0' | jq
func (c *Client) GetInstance(instanceID string) (*Instance, *ErrorResponse, error) {
	i := &Instance{}
	resp, err := c.DoRequest("GET", fmt.Sprintf("/services/%s/%s/instances/%s", c.service, c.environment, instanceID), nil, i)
	if err != nil {
		return nil, resp, err
	}
	return i, resp, nil
}

func (c *Client) GetTask(taskID string) (*Task, error) {
	t := &Task{}
	_, err := c.DoRequest("GET", fmt.Sprintf("/tasks/%s", taskID), nil, t)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (c *Client) WaitForWorkload(taskID string) (string, error) {
	t, err := c.GetTask(taskID)
	if err != nil {
		return "", err
	}
	switch t.Data.Status {
	case "SUCCESS":
		return t.Data.Result.ID, nil
	case "FAILURE":
		return "", fmt.Errorf("provisioning of workload failed")
	default:
		time.Sleep(5 * time.Second)
		return c.WaitForWorkload(taskID)
	}
}

// curl -X 'POST' -d '{"name":"capi-test-jg90","type":"VM","image":"stackpath-edge/centos-7:v202103021226","addAnyCastIpAddress":true,"ports":[{"protocol":"TCP","publicPort":"22"},{"protocol":"TCP","publicPort":"80"}],"firstBootSshKey":"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDgnV5MOhBqpQLt66KGlMKi/VYtmVPUt6epSVxnxrvjayNto5flG2sH4cGqdI2C0NE9/w7BFNdwWqp0mL2kYynC8l+SejW/qjx37hrEBWIXqdTyumchm0LD/7K7P7/kz14IV5NcHjNAsntPgKjx/fzJlbA1VCQYmnOq9RZeKme44rdHYW0BBfgMzekcEbyGTNDGp51NYhVafZLXsF8MzCKlJ+NCPlDqzD6w0fQe/qtMFO8NbFyS9/Lk4prp4HAWEyLSM26w1iLycYpbpWrHw6oc1U7bNIgbsa0ezDu4+OPkxeHz7aG5TeJ/dn0Wftzdfy2sy5PJy5MnYP3RTuROsOv+chu+AshZNNJ9A4ar5gFXSX40sQ0i4GzxZGrsKhW42ZP4sElzV74gEBQ2BOIOJUh4qGRtnjsQCJHBs7DLgpeVeGUq2B7p5zDAlJBGCXiHuTgIM8aVnpdnNrFwmr9SF66iaTrt7x8HinNOCIIztMU15Fk2AYSxSEuju1d3VcPt/d0= jasmingacic@Jasmins-MBP","deployments":[{"name":"wi-peter-qhl","pops":["WAW"],"instancesPerPop":"1"}],"specs":"SP-5"}' -H 'Mc-Api-Key: $TOKEN' 'https://portal.coxedge.com/api/v1/services/edge-services/faefawef/workloads'
func (c *Client) CreateWorkload(data *CreateWorkloadRequest) (*POSTResponse, *ErrorResponse, error) {
	pr := &POSTResponse{}

	data.Name = shortenName(data.Name, 18)

	resp, err := c.DoRequest("POST", fmt.Sprintf("/services/%s/%s/workloads", c.service, c.environment), data, pr)

	return pr, resp, err
}

func (c *Client) DeleteWorkload(workloadID string) (*POSTResponse, *ErrorResponse, error) {
	pr := &POSTResponse{}
	wl, _, err := c.GetWorkload(workloadID)
	if err != nil {
		return nil, nil, err
	}

	resp, err := c.DoRequest("POST", fmt.Sprintf("/services/%s/%s/workloads/%s?operation=delete", c.service, c.environment, workloadID), wl.Data, pr)

	return pr, resp, err
}

func (c *Client) UpdateWorkload(workloadID string, workload WorkloadData) (*POSTResponse, *ErrorResponse, error) {
	pr := &POSTResponse{}
	workload.Name = shortenName(workload.Name, 18)

	resp, err := c.DoRequest("PUT", fmt.Sprintf("/services/%s/%s/workloads/%s", c.service, c.environment, workloadID), workload, pr)

	return pr, resp, err
}

func (c *Client) DoRequest(method, path string, body, v interface{}) (*ErrorResponse, error) {
	req, err := c.NewRequest(method, path, body)
	if err != nil {
		return nil, err
	}

	return c.Do(req, v)
}

func (c *Client) NewRequest(method, path string, body interface{}) (*http.Request, error) {
	// relative path to append to the endpoint url, no leading slash please
	if path[0] == '/' {
		path = path[1:]
	}
	rel, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	u := c.baseUrl.ResolveReference(rel)
	var req *http.Request
	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		req, _ = http.NewRequest(method, u.String(), bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, _ = http.NewRequest(method, u.String(), nil)

	}
	if err != nil {
		return nil, err
	}

	req.Close = true

	req.Header.Add("MC-Api-Key", c.apiKey)
	return req, nil
}

func (c *Client) Do(req *http.Request, v interface{}) (*ErrorResponse, error) {
	if c.debug {
		command, _ := http2curl.GetCurlCommand(req)
		fmt.Println(command)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		o, _ := io.ReadAll(resp.Body)
		errResp := &ErrorResponse{
			StatusCode: resp.StatusCode,
			Errors:     string(o),
		}

		return errResp, fmt.Errorf("%s returned %d", req.URL, resp.StatusCode)
	}
	o, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(o, v)
	if err != nil {
		return nil, err
	}

	return nil, err
}

func shortenName(name string, limit int) string {
	if len(name) <= limit {
		return name
	}

	parts := strings.Split(name, "-")
	var postfix string
	var resize string
	if len(parts) <= 1 {
		resize = strings.Join(parts, "-")
	} else {
		postfix = "-" + parts[len(parts)-1]
		resize = strings.Join(parts[:len(parts)-1], "-")
	}

	trimRange := len(name) - limit
	return resize[:len(resize)-trimRange] + postfix
}

type Workload struct {
	Data WorkloadData `json:"data,omitempty"`
}

type Workloads struct {
	Data []WorkloadData `json:"data,omitempty"`
}
type WorkloadData struct {
	ID                            string                `json:"id"`
	Name                          string                `json:"name"`
	StackID                       string                `json:"stackId"`
	Slug                          string                `json:"slug"`
	Version                       string                `json:"version"`
	Type                          string                `json:"type"`
	Network                       string                `json:"network"`
	CPU                           string                `json:"cpu"`
	Memory                        string                `json:"memory"`
	IsRemoteManagementEnabled     bool                  `json:"isRemoteManagementEnabled"`
	Image                         string                `json:"image"`
	AddImagePullCredentialsOption bool                  `json:"addImagePullCredentialsOption"`
	EnvironmentVariable           []EnvironmentVariable `json:"environmentVariables"`
	SecretEnvironmentVariables    []EnvironmentVariable `json:"secretEnvironmentVariables"`
	AddAnyCastIPAddress           bool                  `json:"addAnyCastIpAddress"`
	AnycastIPAddress              string                `json:"anycastIpAddress"`
	FirstBootSSHKey               string                `json:"firstBootSshKey"`
	Specs                         string                `json:"specs"`
	Deployments                   []Deployment          `json:"deployments"`
	Status                        string                `json:"status"`
	Created                       time.Time             `json:"created"`
	Ports                         []Port                `json:"ports"`
	PersistenceStorageTotalSize   int                   `json:"persistenceStorageTotalSize,omitempty"`
}

type Instances struct {
	Data     []InstanceData `json:"data,omitempty"`
	Metadata Metadata       `json:"metadata,omitempty"`
}

type Instance struct {
	Data InstanceData `json:"data,omitempty"`
}

type CreateWorkloadRequest struct {
	Name                          string                `json:"name,omitempty"`
	Type                          string                `json:"type,omitempty"`
	Image                         string                `json:"image,omitempty"`
	AddImagePullCredentialsOption bool                  `json:"addImagePullCredentialsOption,omitempty"`
	EnvironmentVariables          []EnvironmentVariable `json:"environmentVariables,omitempty"`
	SecretEnvironmentVariables    []EnvironmentVariable `json:"secretEnvironmentVariables,omitempty"`
	AddAnyCastIPAddress           bool                  `json:"addAnyCastIpAddress,omitempty"`
	PersistenceStorageTotalSize   int                   `json:"persistenceStorageTotalSize"`
	Ports                         []Port                `json:"ports,omitempty"`
	FirstBootSSHKey               string                `json:"firstBootSshKey"`
	Deployments                   []Deployment          `json:"deployments,omitempty"`
	Specs                         string                `json:"specs,omitempty"`
	PersistentStorages            []PersistentStorage   `json:"persistentStorages,omitempty"`
	ContainerUsername             string                `json:"containerUsername,omitempty"`
	ContainerPassword             string                `json:"containerPassword,omitempty"`
	ContainerServer               string                `json:"containerServer,omitempty"`
	Commands                      []string              `json:"commands,omitempty"`
	UserData                      string                `json:"userData,omitempty"`
}

type POSTResponse struct {
	TaskId     string `json:"taskId,omitempty"`
	TaskStatus string `json:"taskStatus,omitempty"`
}

type Port struct {
	Protocol       string `json:"protocol"`
	PublicPort     string `json:"publicPort"`
	PublicPortDesc string `json:"publicPortDesc,omitempty"`
}

type EnvironmentVariable struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Deployment defines instance specifications
type Deployment struct {
	// Name of the deployment instance
	Name string `json:"name,omitempty"`
	// CoxEdge PoPs - geographical location for the instance
	Pops []string `json:"pops,omitempty"`
	// +optional
	EnableAutoScaling bool `json:"enableAutoScaling,omitempty"`
	// number of instances per each PoP defined
	// +optional
	InstancesPerPop string `json:"instancesPerPop,omitempty"`
	// +optional
	CPUUtilization int `json:"cpuUtilization,omitempty"`
	// +optional
	MinInstancesPerPop int `json:"minInstancesPerPop,omitempty"`
	// +optional
	MaxInstancesPerPop int `json:"maxInstancesPerPop,omitempty"`
}

type InstanceData struct {
	StackID                   string       `json:"stackId"`
	WorkloadID                string       `json:"workloadId"`
	WorkloadName              string       `json:"workloadName"`
	Name                      string       `json:"name"`
	Type                      string       `json:"type"`
	IPAddress                 string       `json:"ipAddress"`
	PublicIPAddress           string       `json:"publicIpAddress"`
	Location                  string       `json:"location"`
	Created                   time.Time    `json:"created"`
	StartedDate               time.Time    `json:"startedDate"`
	Image                     string       `json:"image"`
	CPU                       string       `json:"cpu"`
	Memory                    string       `json:"memory"`
	EphemeralStorageSize      string       `json:"ephemeralStorageSize"`
	Version                   string       `json:"version"`
	IsRemoteManagementEnabled bool         `json:"isRemoteManagementEnabled"`
	LocationInfo              LocationInfo `json:"locationInfo"`
	InstanceKeyName           string       `json:"instanceKeyName"`
	ID                        string       `json:"id"`
	Status                    string       `json:"status"`
}

type LocationInfo struct {
	City            string  `json:"city"`
	CityCode        string  `json:"cityCode"`
	Subdivision     string  `json:"subdivision"`
	SubdivisionCode string  `json:"subdivisionCode"`
	Country         string  `json:"country"`
	CountryCode     string  `json:"countryCode"`
	Continent       string  `json:"continent"`
	Latitude        float64 `json:"latitude"`
}

type Metadata struct {
	RecordCount int `json:"recordCount"`
}

type ErrorResponse struct {
	StatusCode int
	Errors     string
}

type Task struct {
	Data struct {
		ID      string    `json:"id"`
		Status  string    `json:"status"`
		Created time.Time `json:"created"`
		Result  struct {
			PortRange       string `json:"portRange"`
			Protocol        string `json:"protocol"`
			StackID         string `json:"stackId"`
			WorkloadID      string `json:"workloadId"`
			Description     string `json:"description"`
			Action          string `json:"action"`
			ID              string `json:"id"`
			Source          string `json:"source"`
			Type            string `json:"type"`
			NetworkPolicyID string `json:"networkPolicyId"`
		} `json:"result"`
	} `json:"data"`
}

type PersistentStorage struct {
	Path string `json:"path"`
	Size string `json:"size"`
}
